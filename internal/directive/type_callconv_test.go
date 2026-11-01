package directive

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestParseTypeCallConv(t *testing.T) {
	for _, test := range []struct {
		doc         string
		background  string
		name        string
		replacement string
	}{
		{},
		{doc: "// ordinary comment"},
		{doc: "//llgo:type Python"},
		{doc: "//go:noinline"},
		{doc: "//llgo:cdecl", background: "C", name: "//llgo:cdecl"},
		{doc: "// llgo:cdecl", background: "C", name: "//llgo:cdecl"},
		{doc: "//llgo:stdcall", background: "stdcall", name: "//llgo:stdcall"},
		{doc: "// llgo:stdcall", background: "stdcall", name: "//llgo:stdcall"},
		{doc: "//llgo:type C", background: "C", name: "//llgo:type C", replacement: "//llgo:cdecl"},
		{doc: "// llgo:type C", background: "C", name: "//llgo:type C", replacement: "//llgo:cdecl"},
		{doc: "//llgo:type stdcall", background: "stdcall", name: "//llgo:type stdcall", replacement: "//llgo:stdcall"},
		{doc: "// llgo:type stdcall", background: "stdcall", name: "//llgo:type stdcall", replacement: "//llgo:stdcall"},
	} {
		t.Run(test.doc, func(t *testing.T) {
			fset, doc := typeCallConvDoc(t, test.doc)
			got, err := ParseTypeCallConv(fset, doc)
			if err != nil {
				t.Fatal(err)
			}
			want := TypeCallConv{Background: test.background, Name: test.name, Replacement: test.replacement}
			if test.background != "" {
				want.Pos = doc.List[0].Pos()
			}
			if got != want {
				t.Fatalf("calling convention = %+v, want %+v", got, want)
			}
		})
	}
}

func TestParseTypeCallConvDiagnostics(t *testing.T) {
	for _, test := range []struct {
		doc  string
		want string
	}{
		{"//llgo:cdecl extra", "type.go:2:1: //llgo:cdecl does not accept arguments"},
		{"//llgo:stdcall extra", "type.go:2:1: //llgo:stdcall does not accept arguments"},
		{"//llgo:cdecl\n//llgo:stdcall", "type.go:3:1: conflicting calling-convention directives on one type: //llgo:cdecl and //llgo:stdcall"},
		{"//llgo:cdecl\n//llgo:cdecl", "type.go:3:1: conflicting calling-convention directives on one type: //llgo:cdecl and //llgo:cdecl"},
		{"//llgo:type C\n//llgo:cdecl", "type.go:3:1: conflicting calling-convention directives on one type: //llgo:type C and //llgo:cdecl"},
		{"//llgo:type stdcall\n//llgo:type C", "type.go:3:1: conflicting calling-convention directives on one type: //llgo:type stdcall and //llgo:type C"},
	} {
		t.Run(test.doc, func(t *testing.T) {
			fset, doc := typeCallConvDoc(t, test.doc)
			_, err := ParseTypeCallConv(fset, doc)
			if err == nil || err.Error() != test.want {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func typeCallConvDoc(t *testing.T, comments string) (*token.FileSet, *ast.CommentGroup) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "type.go", "package p\n"+comments+"\ntype T int", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return fset, file.Decls[0].(*ast.GenDecl).Doc
}
