package ssa

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llgo/internal/directive"
)

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
	props, ok := backend.FunctionDirectives(pkg, named.Method(0), original.Decls[1].(*ast.FuncDecl))
	if !ok || props.NoInterface {
		t.Fatal("original source properties were merged with replacement")
	}
}
