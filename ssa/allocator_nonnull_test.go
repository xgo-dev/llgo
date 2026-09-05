//go:build !llgo

package ssa

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestAllocatorNonNullAcrossModules(t *testing.T) {
	Initialize(InitAllTargets | InitAllTargetInfos | InitAllTargetMCs | InitAllAsmPrinters)
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "amd64", LLVMTarget: "x86_64-unknown-linux"},
		{GOOS: "darwin", GOARCH: "arm64", LLVMTarget: "arm64-apple-macosx"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm64-unknown-emscripten"},
	} {
		t.Run(target.LLVMTarget, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			kind := llvm.AttributeKindID("nonnull")
			for _, module := range []string{PkgRuntime, "example.com/caller"} {
				pkg := prog.NewPackage("p", module)
				for _, name := range []string{"AllocU", "AllocZ", "AllocRoot", "Nullable"} {
					fn := pkg.NewFunc(PkgRuntime+"."+name, prog.tyMalloc(), InGo)
					want := name != "Nullable"
					if got := !fn.impl.GetEnumAttributeAtIndex(0, kind).IsNil(); got != want {
						t.Fatalf("%s in %s: nonnull = %v, want %v", name, module, got, want)
					}
					if module == PkgRuntime {
						body := fn.MakeBody(1)
						body.Return(pkg.moduleZeroSizedAlloc(prog.Byte()))
						body.EndBuild()
						continue
					}
					// The caller module has no allocator body and no call-site
					// attributes. Its optimizer must use the external declaration.
					results := types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool]))
					sig := types.NewSignatureType(nil, nil, nil, prog.tyMalloc().Params(), results, false)
					caller := pkg.NewFunc("check"+name, sig, InGo)
					body := caller.MakeBody(1)
					ptr := body.Call(fn.Expr, caller.Param(0))
					if !ptr.impl.GetCallSiteEnumAttribute(0, kind).IsNil() {
						t.Fatal("allocation unexpectedly carries call-site nonnull")
					}
					body.Return(body.BinOp(token.EQL, ptr, prog.Nil(prog.VoidPtr())))
					body.EndBuild()
				}
				mod := pkg.Module()
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				if module == PkgRuntime {
					continue
				}
				pbo := llvm.NewPassBuilderOptions()
				defer pbo.Dispose()
				if err := mod.RunPasses("function(instcombine)", prog.TargetMachine(), pbo); err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"AllocU", "AllocZ", "AllocRoot", "Nullable"} {
					ir := mod.NamedFunction("check" + name).String()
					if got := strings.Contains(ir, "icmp eq ptr"); got != (name == "Nullable") {
						t.Fatalf("unexpected null check after instcombine:\n%s", ir)
					}
					if name != "Nullable" && !strings.Contains(ir, "ret i1 false") {
						t.Fatalf("allocator null check was not folded to false:\n%s", ir)
					}
				}
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
