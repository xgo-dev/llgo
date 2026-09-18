package types_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func BenchmarkTypeCheckSmallPackage(b *testing.B) {
	const src = `package bench

import "example.org/fixture/fmt"

type Number interface{
    ~int | ~float64
}

func twice[T Number](v T) T {
    return v + v
}

func run() {
    fmt.Println(twice(21), twice(1.5))
}`

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "bench.go", src, parser.AllErrors)
		if err != nil {
			b.Fatalf("parse failed: %v", err)
		}

		info := types.Info{
			Types:      make(map[ast.Expr]types.TypeAndValue),
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Implicits:  make(map[ast.Node]types.Object),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
			Scopes:     make(map[ast.Node]*types.Scope),
			Instances:  make(map[*ast.Ident]types.Instance),
		}

		conf := types.Config{Importer: testImporter(b)}
		if _, err := conf.Check("bench", fset, []*ast.File{file}, &info); err != nil {
			b.Fatalf("type-check failed: %v", err)
		}
	}
}
