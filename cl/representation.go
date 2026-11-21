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
	"go/token"
	"go/types"

	llssa "github.com/xgo-dev/llgo/ssa"
)

// representationResolver belongs to the frontend. Definition edges exist only
// while binding attributes; backend Programs receive the resulting snapshot.
type representationResolver struct {
	hasDeclarations bool
	prog            llssa.Program
	pkg             *types.Package
	fset            *token.FileSet
	info            *types.Info
	rhs             map[token.Pos]ast.Expr
	complete        bool
	resolved        map[*types.TypeName]llssa.TypeRepresentation
}

func newRepresentationResolver(prog llssa.Program, fset *token.FileSet, pkg *types.Package, files []*ast.File, info *types.Info) *representationResolver {
	r := &representationResolver{prog: prog, pkg: pkg, fset: fset, info: info, rhs: make(map[token.Pos]ast.Expr)}
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			if spec, ok := node.(*ast.TypeSpec); ok {
				r.hasDeclarations = true
				switch ast.Unparen(spec.Type).(type) {
				case *ast.Ident, *ast.SelectorExpr, *ast.IndexExpr, *ast.IndexListExpr:
					r.rhs[spec.Name.Pos()] = spec.Type
				}
			}
			return true
		})
	}
	return r
}

func registerTypeRepresentations(prog llssa.Program, fset *token.FileSet, pkg *types.Package, files []*ast.File, info *types.Info) error {
	r := newRepresentationResolver(prog, fset, pkg, files, info)
	if !r.hasDeclarations && !pkg.Complete() {
		// Synthetic package owners used by SSA clients can have no source
		// type declarations and no checker phase to finish.
		prog.PublishTypeRepresentations(pkg, nil)
		return nil
	}
	prog.SetTypeRepresentationChecker(pkg, func(obj *types.TypeName) llssa.TypeRepresentation {
		rep, err := r.resolve(obj, nil)
		if err != nil {
			panic(err)
		}
		return rep
	})
	if pkg.Complete() {
		return CompleteTypeRepresentations(prog, fset, pkg, files, info)
	}
	return nil
}

// CompleteTypeRepresentations finishes the frontend's representation phase.
// Call after each successful type check, before dependencies are imported or
// SSA/LLVM lowering starts. Resolution errors publish nothing. The same entry
// point handles one-shot clients whose packages were checked before parsing
// directives; only that frontend path may use CheckExpr to recover bindings.
func CompleteTypeRepresentations(prog llssa.Program, fset *token.FileSet, pkg *types.Package, files []*ast.File, info *types.Info) error {
	if !prog.TypeRepresentationsPending(pkg) {
		return nil
	}
	if !pkg.Complete() {
		return fmt.Errorf("cannot complete representations of unchecked package %s", pkg.Path())
	}
	r := newRepresentationResolver(prog, fset, pkg, files, info)
	r.complete = true
	r.resolved = make(map[*types.TypeName]llssa.TypeRepresentation)
	resolve := func(obj types.Object) error {
		name, ok := obj.(*types.TypeName)
		if !ok {
			return nil
		}
		switch name.Type().(type) {
		case *types.Named, *types.Alias:
			_, err := r.resolve(name, nil)
			return err
		}
		return nil
	}
	if info != nil {
		for _, obj := range info.Defs {
			if err := resolve(obj); err != nil {
				return err
			}
		}
	}
	// Package scope also covers source-less imported objects. Retained lexical
	// scopes recover local declarations for one-shot clients without types.Info.
	var visit func(*types.Scope) error
	visit = func(scope *types.Scope) error {
		for _, name := range scope.Names() {
			if err := resolve(scope.Lookup(name)); err != nil {
				return err
			}
		}
		if info == nil {
			for i := 0; i < scope.NumChildren(); i++ {
				if err := visit(scope.Child(i)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := visit(pkg.Scope()); err != nil {
		return err
	}
	prog.PublishTypeRepresentations(pkg, r.resolved)
	return nil
}

func (r *representationResolver) resolve(obj *types.TypeName, path []*types.TypeName) (rep llssa.TypeRepresentation, err error) {
	if obj.Pkg() != r.pkg {
		return r.prog.RepresentationOf(obj.Type()), nil
	}
	if rep, ok := r.resolved[obj]; ok {
		return rep, nil
	}
	if r.complete {
		defer func() {
			if err == nil {
				r.resolved[obj] = rep
			}
		}()
	}
	if rep = r.prog.DeclaredTypeRepresentation(obj); rep.Storage == llssa.NativeRepresentation {
		return rep, nil
	}
	for _, prev := range path {
		if prev == obj {
			if r.complete {
				return rep, fmt.Errorf("cyclic representation definition for %s", obj)
			}
			return rep, nil // The checker diagnoses invalid source cycles.
		}
	}
	var rhs types.Type
	if expr := r.rhs[obj.Pos()]; expr != nil {
		if r.info != nil {
			rhs = r.info.TypeOf(expr)
			if rhs == nil && r.complete {
				return rep, fmt.Errorf("resolve representation of %s: missing RHS binding", obj)
			}
		} else if r.pkg.Complete() {
			bound := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
			if err := types.CheckExpr(r.fset, r.pkg, expr.Pos(), expr, bound); err != nil {
				return rep, fmt.Errorf("resolve representation of %s: %w", obj, err)
			}
			rhs = bound.TypeOf(expr)
		}
	} else if alias, ok := obj.Type().(*types.Alias); ok {
		rhs = alias.Rhs()
	}
	var next *types.TypeName
	switch t := rhs.(type) {
	case *types.Named:
		next = t.Origin().Obj()
	case *types.Alias:
		next = t.Origin().Obj()
	}
	if next != nil {
		return r.resolve(next, append(path, obj))
	}
	return rep, nil
}
