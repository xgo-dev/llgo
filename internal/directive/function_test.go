package directive

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestFunctionAttributes(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"// llgo:cold\n//llgo:noreturn\n//llgo:cold\nfunc F()", ""},
		{"//llgo:cold\nvar x int", "requires a named function"},
		{"//llgo:cold\nfunc init() {}", "requires a named function"},
		{"//llgo:noreturn\nfunc _() {}", "requires a named function"},
		{"//llgo:cold(x)\nfunc F()", "takes no arguments"},
		{"//llgo:noreturn x\nfunc F()", "takes no arguments"},
		{"//llgo:param(0) unknown\nfunc F(p *int) *int", "unsupported attribute"},
	} {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "attrs.go", "package p\n"+tc.source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		index := new(Index)
		record := index.File(file)
		err = record.ValidateFunctionAttributes(fset)
		if tc.want != "" {
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "attrs.go:2:") {
				t.Fatalf("%s: %v", tc.source, err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		fn := file.Decls[0].(*ast.FuncDecl)
		fn.Doc = nil
		file.Comments = nil
		index.Freeze()
		got := index.Function(fn)
		if !got.Cold || !got.NoReturn {
			t.Fatal("lost prepared attributes after removing comments")
		}
	}
}
