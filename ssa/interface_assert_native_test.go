package ssa

import (
	"go/importer"
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestNativeFunctionCommaOKAssertion(t *testing.T) {
	Initialize(InitAll)
	for _, target := range []*Target{
		{GOOS: "wasip1", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-wasip1", WasmProfile: "w32"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-emscripten", WasmProfile: "j32"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm64-unknown-emscripten", WasmProfile: "j64"},
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "arm64"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH+"/"+target.WasmProfile, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
			prog.SetRuntime(func() *types.Package {
				runtime, err := importer.For("source", nil).Import(PkgRuntime)
				if err != nil {
					t.Fatal(err)
				}
				return runtime
			})
			pkg := prog.NewPackage("p", "example.com/assert")
			rawPkg := types.NewPackage(pkg.Path(), "p")
			callback := types.NewNamed(types.NewTypeName(token.NoPos, rawPkg, "Native", nil),
				types.NewSignatureType(nil, nil, nil, nil, nil, false), nil)
			prog.SetTypeCallConv(rawPkg.Path()+".Native", CDecl)
			emptyIface := types.NewInterfaceType(nil, nil).Complete()
			sig := types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(token.NoPos, nil, "value", emptyIface)),
				types.NewTuple(
					types.NewVar(token.NoPos, nil, "", callback),
					types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool]),
				), false)
			fn := pkg.NewFunc("assertNative", sig, GoDecl)
			b := fn.MakeBody(1)
			asserted := prog.Type(callback, GoDecl)
			pair := b.TypeAssert(fn.Param(0), asserted, true)
			value := b.Extract(pair, 0)
			if value.impl.Type() != asserted.ll {
				t.Fatalf("asserted value has LLVM type %s, want %s", value.impl.Type(), asserted.ll)
			}
			b.Return(value, b.Extract(pair, 1))
			b.EndBuild()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
