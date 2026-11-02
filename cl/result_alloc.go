/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cl

import (
	"go/token"
	"go/types"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// canStackAllocateResult refines SSA's conservative Heap flag for named result
// slots. These slots are allocated once per invocation, so promotion cannot
// merge distinct loop allocations. Leave the shared SSA graph unchanged.
func (p *context) canStackAllocateResult(alloc *ssa.Alloc) bool {
	fn := alloc.Parent()
	if !alloc.Heap || fn == nil || fn.Recover != nil || alloc.Block() != fn.Blocks[0] {
		return false
	}
	results := fn.Signature.Results()
	for i := 0; i < results.Len(); i++ {
		result := results.At(i)
		if result.Name() != "" && result.Name() == alloc.Comment && result.Pos() == alloc.Pos() {
			elem := alloc.Type().Underlying().(*types.Pointer).Elem()
			if p.prog.SizeOf(p.type_(elem, llssa.GoDecl)) == 0 {
				// Preserve the shared sentinel for zero-sized heap allocations.
				return false
			}
			if p.prog.GCRootsEnabled() && mayContainGCRoot(elem) {
				// Explicit roots retain heap objects through their addresses;
				// they do not scan pointer fields in stack storage.
				return false
			}
			analysis := resultEscapeAnalysis{context: p, pkg: fn.Pkg, params: make(map[*ssa.Parameter]bool), budget: 1024}
			return analysis.borrowed(alloc, make(map[ssa.Value]bool))
		}
	}
	return false
}

type resultEscapeAnalysis struct {
	context *context
	pkg     *ssa.Package
	params  map[*ssa.Parameter]bool
	budget  int
}

// borrowed accepts only reads, writes through the address, pointer views, and
// synchronous calls to inspectable Go helpers in the same package. Storing or
// returning the address, closures, go/defer calls, integer conversions, and
// unknown operations all retain heap allocation.
func (a *resultEscapeAnalysis) borrowed(value ssa.Value, seen map[ssa.Value]bool) bool {
	if seen[value] {
		return true
	}
	seen[value] = true
	refs := value.Referrers()
	if refs == nil {
		return false
	}
	for _, ref := range *refs {
		a.budget--
		if a.budget < 0 {
			return false
		}
		switch ref := ref.(type) {
		case *ssa.DebugRef:
		case *ssa.Store:
			if ref.Val == value || ref.Addr != value {
				return false
			}
		case *ssa.UnOp:
			if ref.Op != token.MUL {
				return false
			}
		case *ssa.BinOp:
			if ref.Op != token.EQL && ref.Op != token.NEQ {
				return false
			}
		case *ssa.FieldAddr:
			if !a.borrowed(ref, seen) {
				return false
			}
		case *ssa.IndexAddr:
			if ref.X != value || !a.borrowed(ref, seen) {
				return false
			}
		case *ssa.ChangeType:
			if !a.borrowed(ref, seen) {
				return false
			}
		case *ssa.Convert:
			_, pointer := ref.Type().Underlying().(*types.Pointer)
			if !pointer && basicKind(ref.Type()) != types.UnsafePointer || !a.borrowed(ref, seen) {
				return false
			}
		case *ssa.Phi:
			if !a.borrowed(ref, seen) {
				return false
			}
		case *ssa.Call:
			callee := ref.Call.StaticCallee()
			if callee == nil || callee.Pkg != a.pkg || len(callee.Blocks) == 0 {
				return false
			}
			_, _, kind := a.context.funcName(callee)
			if kind != goFunc {
				return false
			}
			for i, arg := range ref.Call.Args {
				if arg != value {
					continue
				}
				if i >= len(callee.Params) || !a.borrowedParam(callee.Params[i]) {
					return false
				}
			}
		default:
			return false
		}
	}
	return true
}

func (a *resultEscapeAnalysis) borrowedParam(param *ssa.Parameter) bool {
	if safe, known := a.params[param]; known {
		return safe
	}
	// Recursive calls stay conservative instead of assuming their summary.
	a.params[param] = false
	safe := a.borrowed(param, make(map[ssa.Value]bool))
	a.params[param] = safe
	return safe
}
