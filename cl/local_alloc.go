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

// canStackAllocateLocal refines SSA's conservative Heap flag for temporary
// storage whose address is only borrowed. Restrict promotion to the entry block:
// reusing a slot must not merge allocations from different loop iterations.
// Leave the shared SSA graph unchanged.
func (p *context) canStackAllocateLocal(alloc *ssa.Alloc) bool {
	fn := alloc.Parent()
	if !alloc.Heap || fn == nil || fn.Recover != nil || len(fn.Blocks) == 0 || alloc.Block() != fn.Blocks[0] {
		return false
	}
	// Named result slots retain their dedicated escape analysis. Ordinary local
	// allocation must not change the established result allocation semantics.
	if isNamedResultAlloc(alloc) {
		return false
	}
	pointer, ok := alloc.Type().Underlying().(*types.Pointer)
	if !ok {
		return false
	}
	elem := pointer.Elem()
	if p.prog.SizeOf(p.type_(elem, llssa.GoDecl)) == 0 {
		// Preserve the existing shared sentinel for zero-sized heap objects.
		return false
	}
	if p.prog.GCRootsEnabled() && mayContainGCRoot(elem) {
		// Explicit roots retain heap objects, not pointer fields on the stack.
		return false
	}
	if p.borrowedParams == nil {
		p.borrowedParams = make(map[*ssa.Parameter]bool)
	}
	a := localEscapeAnalysis{context: p, pkg: fn.Pkg, params: p.borrowedParams, budget: 1024}
	return a.borrowed(alloc)
}

func isNamedResultAlloc(alloc *ssa.Alloc) bool {
	fn := alloc.Parent()
	if fn == nil {
		return false
	}
	results := fn.Signature.Results()
	for i := 0; i < results.Len(); i++ {
		result := results.At(i)
		if result.Name() != "" && result.Name() == alloc.Comment && result.Pos() == alloc.Pos() {
			return true
		}
	}
	return false
}

type borrowSummary struct {
	param *ssa.Parameter
	safe  bool
	deps  []*ssa.Parameter
}

type localEscapeAnalysis struct {
	context *context
	pkg     *ssa.Package
	params  map[*ssa.Parameter]bool
	budget  int
	graph   map[*ssa.Parameter]*borrowSummary
	pending []*ssa.Parameter
}

// borrowed computes package-local parameter summaries without treating a
// recursive back edge as either an escape or a completed proof. Every address
// use in the reachable call graph must be supported; an escape is propagated
// through all dependent summaries before any result is cached.
func (a *localEscapeAnalysis) borrowed(value ssa.Value) bool {
	if a.params == nil {
		a.params = make(map[*ssa.Parameter]bool)
	}
	a.graph = make(map[*ssa.Parameter]*borrowSummary)
	a.pending = nil
	root := &borrowSummary{}
	root.safe = a.inspect(value, make(map[ssa.Value]bool), root)
	if !root.safe {
		return false
	}
	for i := 0; i < len(a.pending); i++ {
		param := a.pending[i]
		node := a.graph[param]
		node.safe = a.inspect(param, make(map[ssa.Value]bool), node)
		if a.budget < 0 {
			// Never publish incomplete optimistic summaries.
			return false
		}
	}
	reverse := make(map[*ssa.Parameter][]*borrowSummary)
	var unsafe []*ssa.Parameter
	for param, node := range a.graph {
		if !node.safe {
			unsafe = append(unsafe, param)
		}
		for _, dep := range node.deps {
			reverse[dep] = append(reverse[dep], node)
		}
	}
	// A node can be invalidated only once. Iterate until every caller of an
	// unsafe parameter, including callers in the same cycle, is unsafe too.
	for len(unsafe) != 0 {
		param := unsafe[len(unsafe)-1]
		unsafe = unsafe[:len(unsafe)-1]
		for _, caller := range reverse[param] {
			if caller.safe {
				caller.safe = false
				unsafe = append(unsafe, caller.param)
			}
		}
	}
	for param, node := range a.graph {
		a.params[param] = node.safe
	}
	for _, dep := range root.deps {
		if !a.params[dep] {
			return false
		}
	}
	return true
}

// inspect follows address aliases, but never the values loaded from storage.
// A synchronous Go call borrows an address only if all matching parameters
// borrow it too. Unknown uses, stores of the address, and deferred/asynchronous
// calls retain heap storage.
func (a *localEscapeAnalysis) inspect(value ssa.Value, seen map[ssa.Value]bool, summary *borrowSummary) bool {
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
			if !a.inspect(ref, seen, summary) {
				return false
			}
		case *ssa.IndexAddr:
			if ref.X != value || !a.inspect(ref, seen, summary) {
				return false
			}
		case *ssa.ChangeType:
			if !a.inspect(ref, seen, summary) {
				return false
			}
		case *ssa.Convert:
			_, pointer := ref.Type().Underlying().(*types.Pointer)
			if (!pointer && basicKind(ref.Type()) != types.UnsafePointer) || !a.inspect(ref, seen, summary) {
				return false
			}
		case *ssa.Phi:
			if !a.inspect(ref, seen, summary) {
				return false
			}
		case *ssa.Call:
			callee := ref.Call.StaticCallee()
			if callee == nil || callee.Pkg != a.pkg || len(callee.Blocks) == 0 {
				return false
			}
			// Linknames and intrinsics need not execute their visible body.
			if a.context == nil {
				return false
			}
			_, _, kind := a.context.funcName(callee)
			if kind != goFunc {
				return false
			}
			argument := false
			for i, arg := range ref.Call.Args {
				if arg != value {
					continue
				}
				argument = true
				if i >= len(callee.Params) || !a.dependOn(callee.Params[i], summary) {
					return false
				}
			}
			if !argument {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (a *localEscapeAnalysis) dependOn(param *ssa.Parameter, summary *borrowSummary) bool {
	if safe, known := a.params[param]; known {
		return safe
	}
	if _, known := a.graph[param]; !known {
		a.graph[param] = &borrowSummary{param: param}
		a.pending = append(a.pending, param)
	}
	summary.deps = append(summary.deps, param)
	return true
}
