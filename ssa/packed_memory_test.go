//go:build !llgo

package ssa

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestPackedMemoryAlignment(t *testing.T) {
	runtimePkg, err := importer.For("source", nil).Import(PkgRuntime)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "wasip1", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-wasip1", WasmProfile: "w32", WasmProvider: "wasi"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-emscripten", WasmProfile: "j32", WasmProvider: "emscripten"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm64-unknown-emscripten", WasmProfile: "j64", WasmProvider: "emscripten"},
	} {
		name := target.GOARCH
		if target.WasmProfile != "" {
			name += "/" + target.WasmProfile
		}
		t.Run(name, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			prog.SetRuntime(runtimePkg)
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
				types.NewParam(token.NoPos, nil, "", types.NewPointer(types.NewPointer(types.Typ[types.Byte]))),
				types.NewParam(token.NoPos, nil, "", types.NewPointer(types.NewMap(types.Typ[types.Int], types.Typ[types.Int]))),
				types.NewParam(token.NoPos, nil, "", types.NewPointer(types.NewChan(types.SendRecv, types.Typ[types.Int]))),
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
			for _, setOrdering := range []func(Expr, AtomicOrdering) Expr{b.SetOrdering, Expr.SetOrdering} {
				for i := 1; i < params.Len(); i++ {
					x := b.Load(fn.Param(i))
					if x.memoryInstruction().IsALoadInst().IsNil() {
						t.Fatalf("%s: storage casts hide the underlying load", x.RawType())
					}
					x = setOrdering(x, OrderingSeqConsistent)
					store = setOrdering(b.Store(fn.Param(i), x), OrderingSeqConsistent)
					alignment := prog.GoWordSize()
					if i == 1 {
						alignment = 8 // i64 atomics require 8 even on 386.
					}
					for _, inst := range []Expr{x, store} {
						memory := inst.memoryInstruction()
						if got := memory.Alignment(); got != alignment {
							t.Fatalf("%s atomic access: alignment = %d, want %d", x.RawType(), got, alignment)
						}
						if memory.Ordering() != OrderingSeqConsistent {
							t.Fatalf("%s: load/store lost atomic ordering", x.RawType())
						}
					}
					if got := setOrdering(b.Load(fn.Param(i)), OrderingNotAtomic).memoryInstruction().Alignment(); got != 1 {
						t.Fatalf("non-atomic access: alignment = %d, want 1", got)
					}
				}
			}
			b.Return()
			b.EndBuild()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
