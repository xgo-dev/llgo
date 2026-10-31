//go:build !llgo

package ssa

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestPackedMemoryAlignment(t *testing.T) {
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"},
	} {
		t.Run(target.GOARCH, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "packed.go", "package p\n//llgo:packed\ntype Packed struct { B byte; X uint64 }", parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			if err := prog.ParseLayoutSyntax(fset, types.NewPackage("p", "p"), []*ast.File{file}, nil); err != nil {
				t.Fatal(err)
			}
			_, vector := layoutTestVector()
			params := types.NewTuple(
				types.NewParam(token.NoPos, nil, "", types.NewPointer(vector)),
				types.NewParam(token.NoPos, nil, "", types.NewPointer(types.Typ[types.Uint64])),
			)
			pkg := prog.NewPackage("p", "p")
			fn := pkg.NewFunc("p.memory", types.NewSignatureType(nil, nil, nil, params, nil, false), GoDecl)
			b := fn.MakeBody(1)
			defer b.Dispose()
			v := b.Load(fn.Param(0))
			store := b.Store(fn.Param(0), v)
			for _, inst := range []Expr{v, store} {
				if got := inst.memoryInstruction().Alignment(); got != 1 {
					t.Fatalf("possibly unaligned SIMD access: alignment = %d, want 1", got)
				}
			}
			x := b.Load(fn.Param(1)).SetOrdering(OrderingSeqConsistent)
			store = b.Store(fn.Param(1), x).SetOrdering(OrderingSeqConsistent)
			for _, inst := range []Expr{x, store} {
				if got := inst.memoryInstruction().Alignment(); got != 8 {
					t.Fatalf("i64 atomic access: alignment = %d, want 8", got)
				}
			}
			b.Return()
			b.EndBuild()
		})
	}
}
