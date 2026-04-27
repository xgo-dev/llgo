package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"
)

func TestCollectGoCgoPragmas(t *testing.T) {
	src := `package main

// ordinary comment without pragmas
//go:cgo_ldflag "-framework" "CoreFoundation"
/*
 * go:cgo_import_dynamic local alias
 */
func f() {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	ldflags, dynimports := collectGoCgoPragmas([]*ast.File{file})
	if want := []string{"-framework", "CoreFoundation"}; !reflect.DeepEqual(ldflags, want) {
		t.Fatalf("ldflags = %#v, want %#v", ldflags, want)
	}
	if want := []cgoImportDynamicDecl{{local: "local", alias: "alias"}}; !reflect.DeepEqual(dynimports, want) {
		t.Fatalf("dynimports = %#v, want %#v", dynimports, want)
	}
}
