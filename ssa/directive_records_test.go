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

package ssa

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llgo/internal/directive"
)

func TestPackageDirectiveRecordsAreAuthoritative(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	// These entries may come from another package instance with the same path,
	// or from a caller of the compatibility APIs.
	prog.SetLinkname("example.com/p.F", "C.f")
	prog.SetLinkname("example.com/p.V", "C.v")
	prog.SetTypeBackground("example.com/p.T", InC)
	for _, mode := range []string{"no directives", "missing declarations", "no package records"} {
		t.Run(mode, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "p.go", "package p\nfunc F() {}\nvar V int\ntype T struct{}\n", 0)
			if err != nil {
				t.Fatal(err)
			}
			info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
			pkg, err := new(types.Config).Check("example.com/p", fset, []*ast.File{file}, info)
			if err != nil {
				t.Fatal(err)
			}
			if mode != "no package records" {
				var files []*directive.File
				if mode == "no directives" {
					files = prog.Directives().Files([]*ast.File{file})
				}
				records := directive.Collect(files, false, false)
				records.Bind(info)
				prog.SetPackageDirectives(pkg, records)
			}
			fallback := mode == "no package records"
			for _, name := range []string{"F", "V"} {
				want := ""
				if fallback {
					want, _ = prog.Linkname(pkg.Path() + "." + name)
				}
				// Both object-based and name-only callers must respect the package.
				for _, obj := range []types.Object{pkg.Scope().Lookup(name), nil} {
					if got, ok := prog.LinknameFor(pkg, obj, pkg.Path()+"."+name); ok != fallback || got != want {
						t.Errorf("LinknameFor(%s, %v) = %q, %v; want %q, %v", name, obj, got, ok, want, fallback)
					}
				}
			}
			want := InGo
			if fallback {
				want = InC
			}
			named := pkg.Scope().Lookup("T").Type().(*types.Named)
			if got, ok := prog.packageSyntax.namedBackground(named); ok != fallback || got != want {
				t.Errorf("namedBackground = %v, %v; want %v, %v", got, ok, want, fallback)
			}
		})
	}
}

func TestEffectiveDirectivePackageForTypesAndMethods(t *testing.T) {
	fset := token.NewFileSet()
	parse := func(src string) *ast.File {
		f, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	original := parse("package p\ntype T struct{}\nfunc (T) M() {}\n")
	replacement := parse("package p\n//llgo:type C\ntype T struct{}\n//go:nointerface\nfunc (T) M() {}\n")
	info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
	pkg, err := new(types.Config).Check("example.com/p", fset, []*ast.File{original}, info)
	if err != nil {
		t.Fatal(err)
	}
	prog := NewProgram(nil)
	defer prog.Dispose()
	orig := directive.Collect(prog.Directives().Files([]*ast.File{original}), false, false)
	orig.Bind(info)
	prog.SetPackageDirectives(pkg, orig)
	effective := types.NewPackage(pkg.Path(), pkg.Name())
	records := directive.Collect(prog.Directives().Files([]*ast.File{original, replacement}), false, false)
	records.Bind(info)
	prog.SetPackageDirectives(effective, records)
	prog.SetDirectivePackage(pkg, effective)
	prog.Directives().Freeze()
	backend := prog.NewBackendProgram()
	defer backend.Dispose()
	named := pkg.Scope().Lookup("T").Type().(*types.Named)
	if bg, ok := backend.packageSyntax.namedBackground(named); !ok || bg != InC {
		t.Fatalf("effective background = %v, %v", bg, ok)
	}
	if !backend.isNoInterfaceMethod(named.Method(0)) {
		t.Fatal("lost replacement nointerface")
	}
	// A function's source properties are independent of the effective ABI view.
	props := backend.FunctionDeclaration(pkg, named.Method(0), original.Decls[1].(*ast.FuncDecl))
	if props == nil || props.NoInterface {
		t.Fatal("original source properties were merged with replacement")
	}
}
