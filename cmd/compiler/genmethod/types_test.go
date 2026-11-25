package genmethod

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestIsGenericMethod(t *testing.T) {
	fset := token.NewFileSet()
	source := `
		package sample
		type Box struct{}
		func (Box) Plain(int) {}
	`
	if SupportsGenericMethods {
		source += `
		func (Box) Generic[T any](T) {}
		`
	}
	file, err := parser.ParseFile(fset, "test.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
	if _, err := new(types.Config).Check("sample", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		got := IsGenericMethod(info.Defs[fn.Name].Type())
		want := SupportsGenericMethods && fn.Name.Name == "Generic"
		if got != want {
			t.Errorf("IsGenericMethod(%s) = %v, want %v", fn.Name.Name, got, want)
		}
	}
}
