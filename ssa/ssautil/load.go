// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file (LICENSES/Go-BSD-3-Clause.txt).

package ssautil

import (
	"go/ast"
	"go/token"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

// BuildPackage builds an SSA program with SSA intermediate
// representation (IR) for all functions of a single package.
//
// It populates pkg by type-checking the specified file syntax trees. All
// dependencies are loaded using the importer specified by tc. BuildPackage then
// constructs an [ssa.Program] with all dependency packages created, and builds
// and returns the SSA package corresponding to pkg.
func BuildPackage(tc *types.Config, fset *token.FileSet, pkg *types.Package, files []*ast.File, mode ssa.BuilderMode) (*ssa.Package, *types.Info, error) {
	if fset == nil {
		panic("no token.FileSet")
	}
	if pkg.Path() == "" {
		panic("package has no import path")
	}

	info := &types.Info{
		Types:        make(map[ast.Expr]types.TypeAndValue),
		Defs:         make(map[*ast.Ident]types.Object),
		Uses:         make(map[*ast.Ident]types.Object),
		Implicits:    make(map[ast.Node]types.Object),
		Instances:    make(map[*ast.Ident]types.Instance),
		Scopes:       make(map[ast.Node]*types.Scope),
		Selections:   make(map[*ast.SelectorExpr]*types.Selection),
		FileVersions: make(map[*ast.File]string),
	}
	if err := types.NewChecker(tc, fset, pkg, info).Files(files); err != nil {
		return nil, nil, err
	}

	prog := ssa.NewProgram(fset, mode)

	created := make(map[*types.Package]bool)
	var createAll func(pkgs []*types.Package)
	createAll = func(pkgs []*types.Package) {
		for _, p := range pkgs {
			if !created[p] {
				created[p] = true
				prog.CreatePackage(p, nil, nil, true)
				createAll(p.Imports())
			}
		}
	}
	createAll(pkg.Imports())

	ssapkg := prog.CreatePackage(pkg, files, info, false)
	ssapkg.Build()
	return ssapkg, info, nil
}
