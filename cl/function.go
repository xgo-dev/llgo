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
	"fmt"
	"go/ast"
	"go/types"
	"strings"
	"sync"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// Function keeps the concrete SSA body with its prepared directives.
// Generic instances use their origin's directives but retain their own SSA.
// Linkname and ExportName select the effective call binding; patches never
// replace the source body's flags, environment ABI, or wasm import.
type Function struct {
	SSA        *ssa.Function
	Directives llssa.FunctionDirectives
}

func sourceDirectives(prog llssa.Program, fn *ssa.Function) llssa.FunctionDirectives {
	if fn == nil {
		return llssa.FunctionDirectives{}
	}
	if origin := fn.Origin(); origin != nil {
		fn = origin
	}
	decl, _ := fn.Syntax().(*ast.FuncDecl)
	if decl == nil {
		// SSA may represent an imported declaration using type information
		// only. Its concrete package view supplies the same prepared value.
		if fn.Synthetic == "from type information" {
			if obj, ok := fn.Object().(*types.Func); ok {
				name, _ := typesFuncName(llssa.PathOf(obj.Pkg()), obj)
				info, _ := prog.FunctionBinding(obj.Pkg(), name)
				return info
			}
		}
		return llssa.FunctionDirectives{}
	}
	info, ok := prog.FunctionDirectives(decl)
	if !ok {
		panic(fmt.Sprintf("function directives not prepared for %s", fn))
	}
	return info
}

// FunctionSet owns the Go function wrappers of one compilation. All source
// snapshots must be published before use. Lazy SSA adapters may be wrapped
// later, but this never parses source or changes an existing wrapper.
// Function values are immutable after publication and shared by analyses and
// isolated package backends. The mutex protects only the pointer-keyed cache.
type FunctionSet struct {
	prog      llssa.Program
	patches   Patches
	mu        sync.RWMutex
	functions map[*ssa.Function]*Function
}

// NewFunctionSet creates the wrapper cache after package-source preparation.
// patches must remain unchanged for the lifetime of the compilation.
func NewFunctionSet(prog llssa.Program, patches Patches) *FunctionSet {
	return &FunctionSet{prog: prog, patches: patches, functions: make(map[*ssa.Function]*Function)}
}

// Prepare materializes wrappers after SSA and method preparation, before
// backend workers start. Functions created later use the same source snapshots.
func (p *FunctionSet) Prepare(prog *ssa.Program) {
	for fn := range ssautil.AllFunctions(prog) {
		p.Function(fn)
	}
}

func (p *FunctionSet) Function(fn *ssa.Function) *Function {
	p.mu.RLock()
	source := p.functions[fn]
	p.mu.RUnlock()
	if source != nil {
		return source
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if source = p.functions[fn]; source != nil {
		return source
	}
	source = &Function{SSA: fn, Directives: sourceDirectives(p.prog, fn)}
	if fn != nil {
		origin := fn
		if f := fn.Origin(); f != nil {
			origin = f
		}
		// Synthetic adapters may expose the adapted method's types.Func.
		// Its link binding belongs to that method, never to the adapter.
		if origin.Synthetic == "" || origin.Synthetic == "from type information" || origin.Syntax() != nil {
			var pkg *types.Package
			var name string
			if obj, ok := origin.Object().(*types.Func); ok {
				pkg = obj.Pkg()
				name, _ = typesFuncName(llssa.PathOf(pkg), obj)
			} else if origin.Pkg != nil {
				pkg = origin.Pkg.Pkg
				name = funcName(pkg, origin, true)
			}
			if pkg != nil {
				bindingPkg := pkg
				if patch, ok := p.patches[llssa.PathOf(pkg)]; ok {
					bindingPkg = patch.Types
				}
				if binding, prepared := p.prog.FunctionBinding(bindingPkg, name); prepared {
					source.Directives.Linkname = binding.Linkname
					source.Directives.ExportName = binding.ExportName
				} else if origin.Syntax() == nil {
					// Export-object-only imports and backend-provided symbols
					// have no source declaration. Retain their explicit bindings.
					source.Directives.Linkname, _ = p.prog.Linkname(name)
				}
			}
		}
		if !source.Directives.UintptrEscapes && isTransparentAdapter(fn) {
			source.Directives.UintptrEscapes = p.inheritsUintptrEscapes(fn)
		}
	}
	p.functions[fn] = source
	return source
}

func (p *context) sourceFunction(fn *ssa.Function) *Function {
	if p.functions == nil {
		p.functions = NewFunctionSet(p.prog, p.patches)
	}
	return p.functions.Function(fn)
}

func (p *FunctionSet) inheritsUintptrEscapes(fn *ssa.Function) bool {
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
		if sourceDirectives(p.prog, fn).UintptrEscapes {
			return true
		}
		// SSA promoted-method wrappers, thunks (method expressions), and
		// bound-method wrappers have no FuncDecl of their own. Only these
		// transparent adapters inherit the target's pragma;
		// an ordinary function that happens to call it must not inherit it.
		if !isTransparentAdapter(fn) {
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

func isTransparentAdapter(fn *ssa.Function) bool {
	return strings.HasPrefix(fn.Synthetic, "wrapper for ") ||
		strings.HasPrefix(fn.Synthetic, "thunk for ") ||
		strings.HasPrefix(fn.Synthetic, "bound method wrapper for ")
}
