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
	"go/types"
	"strings"

	"golang.org/x/tools/go/ssa"
)

// uintptrEscapesRoots preserves the pointer semantics of //go:uintptrescapes
// at both ends of a call. Ordinary uintptr values remain integers, not roots.
// The source pointer must survive evaluation of subsequent arguments, before
// the callee can publish its uintptr parameters. Go/defer argument records and
// variadic backing arrays are scanned heap allocations and retain the values
// while the call is pending; the callee roots cover the executing call.
func uintptrEscapesRoots(fn *ssa.Function) map[ssa.Value]struct{} {
	roots := make(map[ssa.Value]struct{})
	// Repeated calls in one function share immutable directive facts, without
	// introducing mutable state shared by parallel package backends.
	directives := make(map[*ssa.Function]bool)
	hasDirective := func(callee *ssa.Function) bool {
		value, ok := directives[callee]
		if !ok {
			value = hasUintptrEscapesDirective(callee)
			directives[callee] = value
		}
		return value
	}
	if hasDirective(fn) {
		for _, param := range fn.Params {
			if basicKind(param.Type()) == types.Uintptr {
				roots[param] = struct{}{}
			}
		}
	}
	seen := make(map[ssa.Value]bool)
	var source func(ssa.Value)
	source = func(value ssa.Value) {
		if value == nil || seen[value] {
			return
		}
		seen[value] = true
		switch value := value.(type) {
		case *ssa.Convert:
			// Do not chase integer conversions (including truncation). Go's
			// pragma applies to direct unsafe.Pointer-to-uintptr arguments,
			// not pointers recovered through arbitrary integer expressions.
			if basicKind(value.Type()) == types.Uintptr && basicKind(value.X.Type()) == types.UnsafePointer {
				roots[value.X] = struct{}{}
			}
		case *ssa.ChangeType:
			source(value.X)
		case *ssa.Phi:
			for _, edge := range value.Edges {
				source(edge)
			}
		case *ssa.Slice:
			source(value.X)
		case *ssa.Alloc:
			// SSA constructs f(uintptr(p), uintptr(q)) for ...uintptr as
			// stores into a fresh array followed by a slice of that array.
			// This traces that compiler-created backing, not arbitrary saved
			// []uintptr values or stores into a MakeSlice allocation.
			if refs := value.Referrers(); refs != nil {
				for _, ref := range *refs {
					index, ok := ref.(*ssa.IndexAddr)
					if !ok || index.X != value {
						continue
					}
					if uses := index.Referrers(); uses != nil {
						for _, use := range *uses {
							if store, ok := use.(*ssa.Store); ok && store.Addr == index {
								source(store.Val)
							}
						}
					}
				}
			}
		}
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(ssa.CallInstruction)
			// Interface dispatch has no static callee and does not inherit
			// an implementation's pragma, matching Go's call-site contract.
			if !ok || !hasDirective(call.Common().StaticCallee()) {
				continue
			}
			args := call.Common().Args
			for i, arg := range args {
				if basicKind(arg.Type()) == types.Uintptr {
					source(arg)
				} else if call.Common().Signature().Variadic() && i == len(args)-1 {
					if slice, ok := arg.Type().Underlying().(*types.Slice); ok && basicKind(slice.Elem()) == types.Uintptr {
						source(arg)
					}
				}
			}
		}
	}
	return roots
}

func hasUintptrEscapesDirective(fn *ssa.Function) bool {
	seen := make(map[*ssa.Function]bool)
	var visit func(*ssa.Function) bool
	visit = func(fn *ssa.Function) bool {
		if fn == nil || seen[fn] {
			return false
		}
		seen[fn] = true
		// Generic directives belong to the source declaration. Visit it
		// first; an instantiated body may also retain the same syntax.
		if origin := fn.Origin(); origin != nil && visit(origin) {
			return true
		}
		if hasFuncDirective(fn, "go:uintptrescapes") {
			return true
		}
		// SSA promoted-method wrappers, thunks (method expressions), and
		// bound-method wrappers have no FuncDecl of their own. Only these
		// transparent adapters inherit the target's pragma;
		// an ordinary function that happens to call it must not inherit it.
		if !strings.HasPrefix(fn.Synthetic, "wrapper for ") &&
			!strings.HasPrefix(fn.Synthetic, "thunk for ") &&
			!strings.HasPrefix(fn.Synthetic, "bound method wrapper for ") {
			return false
		}
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				if call, ok := instr.(*ssa.Call); ok && visit(call.Call.StaticCallee()) {
					return true
				}
			}
		}
		return false
	}
	return visit(fn)
}
