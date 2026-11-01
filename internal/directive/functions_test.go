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
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestFunctionIndexKeepsPreparedPositiveAndNegativeRecords(t *testing.T) {
	pkg := pragmaPackage(t, `package p
//go:noinline
func F() {}
func G() {}
`)
	f := pkg.Func("F").Syntax().(*ast.FuncDecl)
	g := pkg.Func("G").Syntax().(*ast.FuncDecl)
	files := []*ast.File{{Decls: []ast.Decl{f, g}}}
	records := CollectFunctions(files)
	// Collection takes a snapshot; publication and queries must not reread docs.
	f.Doc = nil
	g.Doc = &ast.CommentGroup{List: []*ast.Comment{{Text: "//go:noinline"}}}
	var index FunctionIndex
	index.Add(records)
	for _, repeat := range []func(){func() {}, func() { index.Add(CollectFunctions(files)) }, func() { index.PrepareSSA(pkg.Prog) }} {
		repeat()
		if !index.Function(pkg.Func("F")).NoInline || index.Function(pkg.Func("G")).NoInline {
			t.Fatal("preparation or query changed the published snapshot")
		}
	}
}

func TestFunctionIndexRequiresPreparationOnlyForSourceDeclarations(t *testing.T) {
	pkg := pragmaPackage(t, `package p
func F() func() { return func() {} }
`)
	var index FunctionIndex
	fn := pkg.Func("F")
	for _, f := range []*ssa.Function{nil, pkg.Prog.NewFunction("synthetic", fn.Signature, "test"), fn.AnonFuncs[0]} {
		if got := index.Function(f); got != (Function{}) {
			t.Errorf("source-less function attributes = %+v", got)
		}
	}
	func() {
		defer func() {
			got := recover()
			if message, ok := got.(string); !ok || !strings.Contains(message, "function directives were not prepared") {
				t.Fatalf("missing preparation panic = %v", got)
			}
		}()
		index.Function(fn)
	}()
	// An unsuccessful lookup must not have parsed or cached the old empty doc.
	fn.Syntax().(*ast.FuncDecl).Doc = &ast.CommentGroup{List: []*ast.Comment{{Text: "//go:noinline"}}}
	index.PrepareSSA(pkg.Prog)
	if !index.Function(fn).NoInline {
		t.Fatal("lookup changed subsequent preparation")
	}
}

func TestFunctionIndexDistinguishesClonedDeclarationsAtSamePosition(t *testing.T) {
	fset := token.NewFileSet()
	first, err := parser.ParseFile(fset, "p.go", "package p\n//go:noinline\nfunc F() {}", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	original := first.Decls[0].(*ast.FuncDecl)
	clone := *original
	clone.Doc = nil
	second := &ast.File{Name: first.Name, Decls: []ast.Decl{&clone}}
	build := func(file *ast.File) *ssa.Function {
		pkg, _, err := ssautil.BuildPackage(&types.Config{}, fset, types.NewPackage("p", "p"), []*ast.File{file}, ssa.SanityCheckFunctions)
		if err != nil {
			t.Fatal(err)
		}
		return pkg.Func("F")
	}
	a, b := build(first), build(second)
	if a.Prog.Fset != b.Prog.Fset || a.Syntax().Pos() != b.Syntax().Pos() {
		t.Fatal("test requires identical FileSet and declaration position")
	}
	var index FunctionIndex
	index.Add(CollectFunctions([]*ast.File{first, second}))
	if !index.Function(a).NoInline || index.Function(b).NoInline {
		t.Fatal("cloned declarations shared attributes through their source position")
	}
}
