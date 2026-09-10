//go:build !llgo

package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func buildIndexIntrinsicInstance(t *testing.T, path string) *ssa.Function {
	t.Helper()
	const source = `package c
type integer interface { ~int }
func Index[T any, I integer](pointer *T, index I) T { return *pointer }
func Use(pointer **byte) *byte { return Index(pointer, 0) }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "c.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg := types.NewPackage(path, "c")
	ssaPkg, _, err := ssautil.BuildPackage(
		&types.Config{}, fset, pkg, []*ast.File{file},
		ssa.SanityCheckFunctions|ssa.InstantiateGenerics,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, block := range ssaPkg.Func("Use").Blocks {
		for _, instruction := range block.Instrs {
			call, ok := instruction.(*ssa.Call)
			if !ok {
				continue
			}
			if callee := call.Common().StaticCallee(); callee != nil {
				origin := callee.Origin()
				if callee.Name() == "Index" || origin != nil && origin.Name() == "Index" {
					return callee
				}
			}
		}
	}
	t.Fatal("generic Index instance not found")
	return nil
}

func TestCLayoutPointerIntrinsicUsesGenericOrigin(t *testing.T) {
	for _, path := range []string{
		"github.com/goplus/lib/c",
		"github.com/xgo-dev/llgo/runtime/internal/clite",
	} {
		fn := buildIndexIntrinsicInstance(t, path)
		if fn.Origin() == nil {
			t.Fatalf("%s Index instance has no generic origin", path)
		}
		if !isCLayoutPointerIntrinsic(fn) {
			t.Fatalf("%s Index instance did not select C storage", path)
		}
	}
	if isCLayoutPointerIntrinsic(buildIndexIntrinsicInstance(t, "example.com/c")) {
		t.Fatal("ordinary linked Index helper unexpectedly selected C storage")
	}
	if isCLayoutPointerIntrinsic(nil) {
		t.Fatal("nil function unexpectedly selected C storage")
	}
}
