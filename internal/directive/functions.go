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

package directive

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"
	"sync"

	"github.com/xgo-dev/llgo/ssa/abi"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// Function keeps the concrete SSA body with its prepared directives.
// Generic instances use their origin's directives but retain their own SSA.
// Linkname and ExportName select the effective call binding; patches never
// replace the source body's flags, environment ABI, or wasm import.
type Function struct {
	SSA        *ssa.Function
	Directives FunctionDirectives
}

func sourceDirectives(sources *functionSources, fn *ssa.Function) FunctionDirectives {
	if fn == nil {
		return FunctionDirectives{}
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
				name, _ := FuncName(abi.PathOf(obj.Pkg()), obj)
				info, _ := sources.binding(obj.Pkg(), name)
				return info
			}
		}
		return FunctionDirectives{}
	}
	info, ok := sources.declaration(decl)
	if !ok {
		panic(fmt.Sprintf("function directives not prepared for %s", fn))
	}
	return info
}

// Functions owns source preparation and Go function wrappers for a compilation. All source
// snapshots must be published before use. Lazy SSA adapters may be wrapped
// later, but this never parses source or changes an existing wrapper.
// Function values are immutable after publication and shared by analyses and
// isolated package backends. The mutex protects only the pointer-keyed cache.
type Functions struct {
	sources   *functionSources
	patches   map[string]*types.Package
	linkname  func(string) (string, bool)
	mu        sync.RWMutex
	functions map[*ssa.Function]*Function
}

// NewFunctions creates the function preparation and wrapping state of a Program.
// linkname supplies explicit bindings for declarations without prepared source.
func NewFunctions(linkname func(string) (string, bool)) *Functions {
	return &Functions{sources: newFunctionSources(), linkname: linkname, functions: make(map[*ssa.Function]*Function)}
}

// ForCompilation shares prepared source but starts a fresh wrapper cache and
// patch selection. Existing compilations and their backends are not changed.
func (p *Functions) ForCompilation(patches map[string]*types.Package) *Functions {
	selected := make(map[string]*types.Package, len(patches))
	for path, pkg := range patches {
		selected[path] = pkg
	}
	return &Functions{sources: p.sources, patches: selected, linkname: p.linkname, functions: make(map[*ssa.Function]*Function)}
}

func (p *Functions) effectivePackage(pkg *types.Package) *types.Package {
	if replacement := p.patches[abi.PathOf(pkg)]; replacement != nil {
		return replacement
	}
	return pkg
}

// ForObject supplies effective declaration attributes to type metadata paths
// that have no SSA body. It uses exactly the same patch selection as Function.
// The bool reports whether the package was prepared, including known absence.
func (p *Functions) ForObject(fn *types.Func) (FunctionDirectives, bool) {
	if fn == nil || fn.Pkg() == nil {
		return FunctionDirectives{}, false
	}
	fn = fn.Origin()
	if recv := fn.Signature().Recv(); recv != nil {
		typ := types.Unalias(recv.Type())
		if ptr, ok := typ.(*types.Pointer); ok {
			typ = types.Unalias(ptr.Elem())
		}
		if _, named := typ.(*types.Named); !named {
			// Type rewriting can reconstruct interface methods with an
			// unnamed receiver. They have no FuncDecl-style receiver name;
			// retain the type-metadata spelling and prepared-absence rule.
			name := abi.PathOf(fn.Pkg()) + "." + types.TypeString(recv.Type(), abi.PathOf) + "." + fn.Name()
			return p.sources.binding(p.effectivePackage(fn.Pkg()), name)
		}
	}
	name, _ := FuncName(abi.PathOf(fn.Pkg()), fn)
	return p.sources.binding(p.effectivePackage(fn.Pkg()), name)
}

// Prepare materializes wrappers after SSA and method preparation, before
// backend workers start. Functions created later use the same source snapshots.
func (p *Functions) Prepare(prog *ssa.Program) {
	for fn := range ssautil.AllFunctions(prog) {
		p.Function(fn)
	}
}

func (p *Functions) Function(fn *ssa.Function) *Function {
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
	source = &Function{SSA: fn, Directives: sourceDirectives(p.sources, fn)}
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
				name, _ = FuncName(abi.PathOf(pkg), obj)
			} else if origin.Pkg != nil {
				pkg = origin.Pkg.Pkg
				name = abi.FullName(pkg, origin.Name())
				if recv := origin.Signature.Recv(); recv != nil {
					canonical, _ := receiverName(recv.Type())
					name = abi.PathOf(pkg) + "." + canonical + "." + origin.Name()
				}
			}
			if pkg != nil {
				if binding, prepared := p.sources.binding(p.effectivePackage(pkg), name); prepared {
					source.Directives.Linkname = binding.Linkname
					source.Directives.ExportName = binding.ExportName
				} else if origin.Syntax() == nil && p.linkname != nil {
					// Export-object-only imports and backend-provided symbols
					// have no source declaration. Retain their explicit bindings.
					source.Directives.Linkname, _ = p.linkname(name)
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

func (p *Functions) inheritsUintptrEscapes(fn *ssa.Function) bool {
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
		if sourceDirectives(p.sources, fn).UintptrEscapes {
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
