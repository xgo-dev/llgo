// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file (LICENSES/Go-BSD-3-Clause.txt).

package ssautil

import (
	"go/ast"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

// AllFunctions finds and returns the set of functions potentially
// needed by program prog, as determined by a simple linker-style
// reachability algorithm starting from the members and method-sets of
// each package. The result may include anonymous functions and
// synthetic wrappers.
//
// Precondition: all packages are built.
func AllFunctions(prog *ssa.Program) map[*ssa.Function]bool {
	seen := make(map[*ssa.Function]bool)

	var function func(fn *ssa.Function)
	function = func(fn *ssa.Function) {
		if fn != nil && !seen[fn] {
			seen[fn] = true
			var buf [10]*ssa.Value
			for _, b := range fn.Blocks {
				for _, instr := range b.Instrs {
					for _, op := range instr.Operands(buf[:0]) {
						if op != nil {
							if fn, ok := (*op).(*ssa.Function); ok {
								function(fn)
							}
						}
					}
				}
			}
		}
	}

	methodsOf := func(T types.Type) {
		if !types.IsInterface(T) {
			mset := prog.MethodSets.MethodSet(T)
			for i := 0; i < mset.Len(); i++ {
				sel := mset.At(i)
				if sel.Obj().(*types.Func).Signature().TypeParams() == nil {
					function(prog.MethodValue(sel))
				}
			}
		}
	}

	isSyntactic := func(p *ssa.Package) bool {
		if p == nil {
			return false
		}
		init := p.Func("init")
		return init != nil && len(init.Blocks) > 0
	}

	exportedTypeHack := func(t *ssa.Type) {
		if isSyntactic(t.Package()) &&
			ast.IsExported(t.Name()) &&
			!types.IsInterface(t.Type()) {
			if named, ok := t.Type().(*types.Named); ok {
				if named.TypeParams() == nil {
					methodsOf(named)
					methodsOf(types.NewPointer(named))
				}
			}
		}
	}

	for _, pkg := range prog.AllPackages() {
		for _, mem := range pkg.Members {
			switch mem := mem.(type) {
			case *ssa.Function:
				function(mem)
			case *ssa.Type:
				exportedTypeHack(mem)
			}
		}
	}

	for _, T := range prog.RuntimeTypes() {
		methodsOf(T)
	}

	return seen
}

// MainPackages returns the subset of the specified packages
// named "main" that define a main function.
func MainPackages(pkgs []*ssa.Package) []*ssa.Package {
	var mains []*ssa.Package
	for _, pkg := range pkgs {
		if pkg != nil && pkg.Pkg.Name() == "main" && pkg.Func("main") != nil {
			mains = append(mains, pkg)
		}
	}
	return mains
}
