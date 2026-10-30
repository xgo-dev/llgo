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
)

func TestFunctionDirectiveAttachmentRules(t *testing.T) {
	tests := []struct {
		doc  string
		want bool
	}{
		{"//go:nointerface", true},
		{"//go:nointerface\n//go:noinline", true},
		{"//go:nointerface\n// unrelated prose", false},
		{"//go:nointerface ", false},
		{"// unrelated prose\n//go:nointerface", true},
	}
	for _, test := range tests {
		t.Run(test.doc, func(t *testing.T) {
			// Construct comments directly: go/parser may strip trailing spaces.
			doc := new(ast.CommentGroup)
			for _, line := range strings.Split(test.doc, "\n") {
				doc.List = append(doc.List, &ast.Comment{Text: line})
			}
			got, err := parseFunction(nil, doc, "T.M", false)
			if err != nil || got.NoInterface != test.want {
				t.Fatalf("nointerface = %v, error = %v; want %v", got.NoInterface, err, test.want)
			}
		})
	}
}

func TestRecvTypeName(t *testing.T) {
	if ret := recvTypeName(&ast.IndexExpr{
		X:     &ast.Ident{Name: "Pointer"},
		Index: &ast.Ident{Name: "T"},
	}); ret != "Pointer" {
		t.Fatal("recvTypeName IndexExpr:", ret)
	}
	if ret := recvTypeName(&ast.IndexListExpr{
		X:       &ast.Ident{Name: "Pointer"},
		Indices: []ast.Expr{&ast.Ident{Name: "T"}},
	}); ret != "Pointer" {
		t.Fatal("recvTypeName IndexListExpr:", ret)
	}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("recvTypeName: no error?")
		}
	}()
	recvTypeName(&ast.BadExpr{})
}

func TestASTFunctionNames(t *testing.T) {
	full, inPkg := astFuncName("example.com/p", &ast.FuncDecl{Name: &ast.Ident{Name: "F"}}, nil)
	if full != "example.com/p.F" || inPkg != "F" {
		t.Fatalf("astFuncName(func)=(%q,%q), want (%q,%q)", full, inPkg, "example.com/p.F", "F")
	}

	ptrRecv := &ast.FuncDecl{
		Name: &ast.Ident{Name: "M"},
		Recv: &ast.FieldList{List: []*ast.Field{
			{Type: &ast.StarExpr{X: &ast.ParenExpr{X: &ast.Ident{Name: "T"}}}},
		}},
	}
	full, inPkg = astFuncName("example.com/p", ptrRecv, nil)
	if full != "example.com/p.(*T).M" || inPkg != "(*T).M" {
		t.Fatalf("astFuncName(method ptr)=(%q,%q), want (%q,%q)", full, inPkg, "example.com/p.(*T).M", "(*T).M")
	}

}

func TestClosureEnvDirectiveCacheUsesSourceIdentity(t *testing.T) {
	functions := NewFunctions(nil)
	pkg := types.NewPackage("example.com/p", "p")
	decl := &ast.FuncDecl{Name: &ast.Ident{Name: "entry", NamePos: 7}}
	info := &FunctionDirectives{ClosureEnv: true}
	functions.Register(pkg, decl, "example.com/p.entry", *info)
	if got, ok := functions.sources.declaration(decl); !ok || !got.ClosureEnv {
		t.Fatal("lost source directive")
	}
	// Same token position and spelling in another AST is a distinct declaration.
	other := &ast.FuncDecl{Name: &ast.Ident{Name: "entry", NamePos: 7}}
	if _, ok := functions.sources.declaration(other); ok {
		t.Fatal("distinct source declarations shared directives")
	}

}

func TestPackagePreparationOwnsSnapshots(t *testing.T) {
	functions := NewFunctions(nil)
	pkg := types.NewPackage("example.com/p", "p")
	fset := token.NewFileSet()
	parse := func(source string) *ast.File {
		file, err := parser.ParseFile(fset, "p.go", source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		return file
	}
	file := parse(`package p
//go:noinline
//go:linkname F C.old
func F() {}
`)
	bindings, err := functions.PreparePackage(fset, pkg, []*ast.File{file}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	bindings["example.com/p.F"] = FunctionDirectives{Linkname: "C.modified"}
	fn := types.NewFunc(token.NoPos, pkg, "F", types.NewSignatureType(nil, nil, nil, nil, nil, false))
	if info, prepared := functions.ForObject(fn); !prepared || info.Linkname != "C.old" || !info.NoInline {
		t.Fatalf("returned linker data mutated prepared source: %+v", info)
	}
	bad := parse(`package p
//go:linkname F C.new
func F() {}
//llgo:link Wrong C.bad
func G() {}
`)
	if _, err := functions.PreparePackage(fset, pkg, []*ast.File{bad}, false, false); err == nil {
		t.Fatal("bad link accepted")
	}
	if info, _ := functions.ForObject(fn); info.Linkname != "C.old" {
		t.Fatal("failed preparation published partial state")
	}
	missing := types.NewFunc(token.NoPos, pkg, "Missing", types.NewSignatureType(nil, nil, nil, nil, nil, false))
	if info, prepared := functions.ForObject(missing); !prepared || info != (FunctionDirectives{}) {
		t.Fatal("known absence lost")
	}
	other := types.NewPackage(pkg.Path(), pkg.Name())
	missing = types.NewFunc(token.NoPos, other, "Missing", types.NewSignatureType(nil, nil, nil, nil, nil, false))
	if _, prepared := functions.ForObject(missing); prepared {
		t.Fatal("same-path package inherited preparation")
	}
}

func TestObjectAttributesForUnnamedInterfaceReceiver(t *testing.T) {
	functions := NewFunctions(nil)
	pkg := types.NewPackage("example.com/interface", "iface")
	recv := types.NewVar(token.NoPos, pkg, "", types.NewInterfaceType(nil, nil).Complete())
	method := types.NewFunc(token.NoPos, pkg, "Read", types.NewSignatureType(recv, nil, nil, nil, nil, false))
	if info, prepared := functions.ForObject(method); prepared || info != (FunctionDirectives{}) {
		t.Fatal("unprepared interface acquired attributes")
	}
	if _, err := functions.PreparePackage(token.NewFileSet(), pkg, nil, false, false); err != nil {
		t.Fatal(err)
	}
	if info, prepared := functions.ForObject(method); !prepared || info != (FunctionDirectives{}) {
		t.Fatal("unnamed interface lost known absence")
	}
}
