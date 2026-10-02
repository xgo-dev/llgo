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
	"reflect"
	"strings"
	"testing"
)

func TestCollectDeclarationContracts(t *testing.T) {
	fset, file := parseSource(t, `package C
//llgo:skip imported
import _ "unsafe"
//export public
func public() {}
func Xautomatic() {}
func Automatic() {}
func private() {}
//llgo:link explicit C.custom
func explicit() {}
//llgo:link exported ignored.symbol
//export exported
func exported() {}
//export ignored
//llgo:link linked C.linked
func linked() {}
//export wrong
func invalid() {}
//llgo:link single attached
var single int
var many, other int
//llgo:type C
type Native func()
//llgo:type stdcall
type (Grouped func(); Another func())
//llgo:skip removed
const keep = 1
//llgo:skipall
type Last int
//go:linkname single final.symbol
//go:linkname absent ignored
`)
	p, err := Collect(fset, new(Index).Files([]*ast.File{file}), true, false)
	if err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]string{"public": "public", "Xautomatic": "automatic", "Automatic": "Automatic", "explicit": "C.custom", "exported": "exported", "linked": "C.linked"} {
		d := p.Names[name].(*FunctionDecl)
		if d.Err != nil || !d.HasLinkname || d.Linkname != target {
			t.Errorf("%s link = %+v", name, d)
		}
	}
	for _, name := range []string{"public", "Xautomatic", "Automatic", "exported"} {
		d := p.Names[name].(*FunctionDecl)
		if d.ExportName != d.Linkname {
			t.Errorf("%s export = %q", name, d.ExportName)
		}
	}
	if d := p.Names["private"].(*FunctionDecl); d.HasLinkname {
		t.Fatal("private function was auto-exported")
	}
	if d := p.Names["invalid"].(*FunctionDecl); d.Err == nil || !strings.Contains(d.Err.Error(), `wrong name "wrong"`) {
		t.Fatalf("deferred export error = %v", d.Err)
	}
	if d := p.Names["single"].(*VariableDecl); d.Linkname != "final.symbol" {
		t.Fatalf("file link did not override attached link: %+v", d)
	}
	if p.Names["many"].(*VariableDecl).HasLinkname || p.Names["other"].(*VariableDecl).HasLinkname {
		t.Fatal("multi-variable declaration consumed an attached link")
	}
	if p.Names["Native"].(*TypeDecl).Background != "C" || p.Names["Grouped"].(*TypeDecl).Background != "" {
		t.Fatal("type background attachment changed")
	}
	if !p.Skip.All || !reflect.DeepEqual(p.Skip.Names, []string{"imported", "removed"}) {
		t.Fatalf("skip = %+v", p.Skip)
	}
	renamedPackage, err := Collect(fset, new(Index).Files([]*ast.File{file}), false, true)
	if err != nil {
		t.Fatal(err)
	}
	renamed := renamedPackage.Names["invalid"].(*FunctionDecl)
	if renamed.Err != nil || renamed.ExportName != "wrong" {
		t.Fatalf("renamed export = %+v", renamed)
	}
}

func TestPackageLinksRequireUnsafe(t *testing.T) {
	fset, file := parseSource(t, `package p
//llgo:link F first
//go:linkname other unrelated
//llgo:link F last
func F() {}

//go:linkname F file.symbol
//go:linkname malformed
`)
	p, err := Collect(fset, new(Index).Files([]*ast.File{file}), false, false)
	if err != nil {
		t.Fatal(err)
	}
	d := p.Names["F"].(*FunctionDecl)
	if d.Linkname != "last" {
		t.Fatalf("link without unsafe = %q", d.Linkname)
	}
}

func checkRecordSource(t *testing.T, fset *token.FileSet, file *ast.File) (*types.Package, *types.Info) {
	t.Helper()
	info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
	pkg, err := new(types.Config).Check("example.com/p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	return pkg, info
}

func TestBindingUsesEffectiveDeclarationsAndPositions(t *testing.T) {
	fset := token.NewFileSet()
	parse := func(name, source string) *ast.File {
		f, err := parser.ParseFile(fset, name, source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	original := parse("original.go", "package p\nfunc F() {}\nvar V int\ntype T int\n")
	replacement := parse("replacement.go", "package p\n//go:noinline\nfunc F() {}\nvar V string\n//llgo:type C\ntype T string\n")
	_, oldInfo := checkRecordSource(t, fset, original)
	newPkg, newInfo := checkRecordSource(t, fset, replacement)
	files := new(Index).Files([]*ast.File{original, replacement})
	p, err := Collect(fset, files, false, false)
	if err != nil {
		t.Fatal(err)
	}
	p.Bind(nil)
	p.Bind(oldInfo)
	if len(p.Objects) != 0 {
		t.Fatalf("inactive originals bound: %v", p.Objects)
	}
	p.Bind(newInfo)
	for _, name := range []string{"F", "V", "T"} {
		if got := p.Objects[newPkg.Scope().Lookup(name)]; got != p.Names[name] {
			t.Errorf("%s binding = %v", name, got)
		}
	}
	if !p.Names["F"].(*FunctionDecl).NoInline || p.Names["T"].(*TypeDecl).Background != "C" {
		t.Fatal("replacement properties lost")
	}
}

func TestParenthesizedGenericReceiverSelector(t *testing.T) {
	expr, err := parser.ParseExpr("(Pair[A, B])")
	if err != nil {
		t.Fatal(err)
	}
	fn := &ast.FuncDecl{Name: ast.NewIdent("M"), Recv: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: expr}}}}}
	if got := FuncName(fn); got != "(*Pair).M" {
		t.Fatalf("selector = %q", got)
	}
}

func TestCollectLinkDiagnostics(t *testing.T) {
	for _, tt := range []struct {
		source string
		want   string
	}{
		{"//llgo:link (Ptr).M C.m\nfunc (Ptr) M() {}", ""},
		{"//llgo:link Wrong.M C.m\nfunc (Ptr) M() {}", `local name "Wrong.M" does not match declaration "Ptr.M"`},
		{"//llgo:link Wrong.M C.m\n//llgo:link Ptr.M C.valid\nfunc (Ptr) M() {}", `local name "Wrong.M" does not match declaration "Ptr.M"`},
		{"//llgo:link Wrong C.f\n//export F\nfunc F() {}", `local name "Wrong" does not match declaration "F"`},
		{"//llgo:link Ptr.M\n//llgo:link Ptr.M C.valid\nfunc (Ptr) M() {}", "requires a local name and a target"},
		{"//llgo:link Ptr.M\nfunc (Ptr) M() {}", "requires a local name and a target"},
		{"func (Ptr) M() {}\n//llgo:link Ptr.M C.m", "is not attached to a declaration"},
		{"func (Ptr) M() {}\n//llgo:link Ptr.M", "requires a local name and a target"},
		{"func (Ptr) M() {}\n//go:linkname Wrong.M C.m", `local method "Wrong.M" not found`},
		{"func (Ptr) M() {}\n//go:linkname (Ptr).M C.m", ""},
		{"//llgo:link v C.v\nvar v int", ""},
		{"//llgo:link other C.v\nvar v int", `local name "other" does not match declaration "v"`},
		{"var (a int; b int)\n//go:linkname a C.a", ""},
		{"//llgo:link a C.a\nvar a, b int", "is not attached to a declaration"},
		{"func F() {}\n//go:linkname F", ""},
	} {
		t.Run(tt.source, func(t *testing.T) {
			fset, file := parseSource(t, "package p\nimport _ \"unsafe\"\ntype T struct{}\ntype Ptr = *T\n"+tt.source+"\n")
			_, err := Collect(fset, new(Index).Files([]*ast.File{file}), false, false)
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), ".go:") {
				t.Fatalf("error = %v, want source position and %q", err, tt.want)
			}
		})
	}
}
