// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package directive

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

const locationDecls = `package location
import _ "unsafe"
//go:linkname FILE llgo.file
var FILE *int8
//go:linkname LINE llgo.line
var LINE int32
`

func checkVariables(t *testing.T, source string) (*token.FileSet, []*ast.File, *types.Package, *types.Info, Variables, error) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "location.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	vars, err := CollectVariables(fset, files, false)
	if err != nil {
		return fset, files, nil, nil, vars, err
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Types: map[ast.Expr]types.TypeAndValue{}}
	pkg, err := (&types.Config{Importer: importer.Default()}).Check("example/location", fset, files, info)
	if err == nil {
		err = vars.Validate(fset, pkg)
	}
	return fset, files, pkg, info, vars, err
}

func TestCompilerVariableDeclarations(t *testing.T) {
	for _, test := range []struct{ name, source, want string }{
		{"valid", locationDecls, ""},
		{"aliases", strings.ReplaceAll(strings.ReplaceAll(locationDecls, "*int8", "*Char"), "int32", "Int") + "type Char = int8; type Int = int32", ""},
		{"named", strings.ReplaceAll(strings.ReplaceAll(locationDecls, "*int8", "*Char"), "int32", "Int") + "type Char int8; type Int int32", ""},
		{"wrong line type", strings.ReplaceAll(locationDecls, "int32", "int"), "must have type int32"},
		{"wrong file type", strings.ReplaceAll(locationDecls, "*int8", "string"), "must have type *int8"},
		{"unsigned char", strings.ReplaceAll(locationDecls, "*int8", "*byte"), "must have type *int8"},
		{"initializer", strings.ReplaceAll(locationDecls, "LINE int32", "LINE int32 = 0"), "cannot have an initializer"},
		{"nil initializer", strings.ReplaceAll(locationDecls, "FILE *int8", "FILE *int8 = nil"), "cannot have an initializer"},
		{"unknown", strings.ReplaceAll(locationDecls, "llgo.line", "llgo.lien"), `unknown compiler variable "llgo.lien"`},
		{"phase two", strings.ReplaceAll(locationDecls, "llgo.file", "llgo.func"), `unknown compiler variable "llgo.func"`},
		{"const", locationDecls + "const N = LINE", "is not constant"},
		{"array", locationDecls + "var A [LINE]int", "invalid array length"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, _, _, _, _, err := checkVariables(t, test.source)
			if test.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestCompilerVariableUses(t *testing.T) {
	for _, test := range []struct{ name, body, want string }{
		{"reads", `func f() { x,y := LINE,FILE; _,_ = x,y; switch x {case LINE:} }`, ""},
		{"shadow", `func f() { LINE := int32(1); LINE++; _ = &LINE; x:=struct{LINE int32}{}; x.LINE=3 }`, ""},
		{"address of copy", `var copy = LINE; var ptr = &copy`, ""},
		{"write", `func f() { LINE = 3 }`, "assign to"},
		{"compound", `func f() { LINE += 1 }`, "assign to"},
		{"increment", `func f() { LINE++ }`, "assign to"},
		{"decrement", `func f() { LINE-- }`, "assign to"},
		{"tuple", `func f() { x:=int32(0); x,LINE=1,2; _=x }`, "assign to"},
		{"range", `func f() { for LINE = range int32(3) {} }`, "assign to"},
		{"range value", `func f() { for _,LINE = range []int32{3} {} }`, "assign to"},
		{"address", `func f() { _ = &FILE }`, "take address of"},
		{"parentheses", `func f() { _ = &((LINE)) }`, "take address of"},
		{"erased address", `func f() int32 { return *(&LINE) }`, "take address of"},
		{"unreachable", `func f() { if false { LINE=1 } }`, "assign to"},
		{"generic", `func f[T any]() { _ = &LINE }`, "take address of"},
		{"package initializer", `var ptr = &LINE`, "take address of"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fset, files, pkg, info, vars, err := checkVariables(t, locationDecls+test.body)
			if err != nil {
				t.Fatal(err)
			}
			lookup := func(v *types.Var) Variable {
				if v.Pkg() == pkg && pkg.Scope().Lookup(v.Name()) == v {
					return vars[v.Name()]
				}
				return Variable{}
			}
			for _, bindings := range []*types.Info{info, nil} {
				err := ValidateVariableUses(fset, pkg, bindings, files, lookup)
				if test.want == "" {
					if err != nil {
						t.Fatal(err)
					}
				} else if err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("info=%v error = %v, want %q", bindings != nil, err, test.want)
				}
			}
		})
	}
}

func TestCompilerVariablePackageLinks(t *testing.T) {
	fset := token.NewFileSet()
	var files []*ast.File
	for name, src := range map[string]string{
		"decl.go":  "package p\nvar (Line int32; File *int8; Ordinary int)\n",
		"links.go": "package p\nimport _ \"unsafe\"\n//go:linkname Line llgo.line\n//go:linkname File llgo.file\n",
	} {
		file, err := parser.ParseFile(fset, name, src, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, file)
	}
	vars, err := CollectVariables(fset, files, false)
	if err != nil {
		t.Fatal(err)
	}
	if vars["Line"].Kind != LineVariable || vars["File"].Kind != FileVariable || vars["Ordinary"].Kind != OrdinaryVariable {
		t.Fatalf("records = %+v", vars)
	}
	// A snapshot retains no comment AST references.
	for _, file := range files {
		for _, group := range file.Comments {
			for _, comment := range group.List {
				comment.Text = "// changed"
			}
		}
	}
	if vars["Line"].Linkname != "llgo.line" {
		t.Fatal("snapshot changed")
	}
}

func TestVariableLinkPrecedence(t *testing.T) {
	for _, test := range []struct {
		name, source, link, export, want string
		rename                           bool
	}{
		{name: "attached LLGo", source: "//llgo:link V C.value\nvar V int32", link: "C.value"},
		{name: "spaced LLGo", source: "// llgo:link V C.value\nvar V int32", link: "C.value"},
		{name: "package Go wins", source: "//llgo:link V C.old\nvar V int32\n//go:linkname V C.new", link: "C.new"},
		{name: "last Go wins", source: "//go:linkname V C.old\n//go:linkname V C.new\nvar V int32", link: "C.new"},
		{name: "export", source: "//export V\nvar V int32", link: "V", export: "V"},
		{name: "empty export", source: "//export\nvar V int32"},
		{name: "link overrides export", source: "//export V\n//llgo:link V C.value\nvar V int32", link: "C.value"},
		{name: "export rename", source: "//export renamed\nvar V int32", link: "renamed", export: "renamed", rename: true},
		{name: "export mismatch", source: "//export other\nvar V int32", want: "export comment has wrong name"},
		{name: "LLGo mismatch", source: "//llgo:link other C.value\nvar V int32", want: "does not match declaration"},
		{name: "LLGo missing target", source: "//llgo:link V\nvar V int32", want: "requires a local name and a target"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "links.go", "package p\nimport _ \"unsafe\"\n"+test.source, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			vars, err := CollectVariables(fset, []*ast.File{file}, test.rename)
			if test.want != "" {
				if err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("error=%v, want %s", err, test.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if rec := vars["V"]; rec.Linkname != test.link || rec.ExportName != test.export {
				t.Fatalf("record=%+v, want link=%s export=%s", rec, test.link, test.export)
			}
		})
	}
}

type variableTestImporter struct{ pkg *types.Package }

func (i variableTestImporter) Import(string) (*types.Package, error) { return i.pkg, nil }

func TestCompilerVariableImportedUses(t *testing.T) {
	imported := types.NewPackage("example/location", "location")
	line := types.NewVar(token.NoPos, imported, "Line", types.Typ[types.Int32])
	imported.Scope().Insert(line)
	imported.MarkComplete()
	for _, body := range []string{"_ = &location.Line", "location.Line = 1"} {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "use.go", "package p\nimport \"example/location\"\nfunc f() { "+body+" }", 0)
		if err != nil {
			t.Fatal(err)
		}
		info := &types.Info{Uses: map[*ast.Ident]types.Object{}}
		files := []*ast.File{file}
		pkg, err := (&types.Config{Importer: variableTestImporter{imported}}).Check("example/p", fset, files, info)
		if err != nil {
			t.Fatal(err)
		}
		lookup := func(v *types.Var) Variable {
			if v != line {
				t.Fatalf("resolved %v, want imported declaration %v", v, line)
			}
			return Variable{Name: "Line", Kind: LineVariable}
		}
		for _, bindings := range []*types.Info{info, nil} {
			err := ValidateVariableUses(fset, pkg, bindings, files, lookup)
			if err == nil || !strings.Contains(err.Error(), "compiler variable Line") {
				t.Fatalf("%s: info=%v error=%v, want rejection of imported compiler variable use", body, bindings != nil, err)
			}
		}
	}
}
