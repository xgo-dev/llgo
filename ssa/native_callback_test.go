package ssa

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestNativeUintptrCallbackAdapters(t *testing.T) {
	for _, profile := range []string{"j32", "j64", "w32"} {
		for _, results := range []int{0, 1, 2} {
			t.Run(fmt.Sprintf("%s/results-%d", profile, results), func(t *testing.T) {
				target := &Target{GOOS: "js", GOARCH: "wasm", WasmProfile: profile, LLVMTarget: "wasm32-unknown-emscripten"}
				if profile == "j64" {
					target.LLVMTarget = "wasm64-unknown-emscripten"
				} else if profile == "w32" {
					target.GOOS, target.LLVMTarget = "wasip1", "wasm32-unknown-wasi"
				}
				prog := NewProgram(target)
				defer prog.Dispose()
				setTestRuntime(t, prog)
				pkg := prog.NewPackage("p", "example.com/p")
				param := types.NewParam(token.NoPos, nil, "n", types.Typ[types.Uintptr])
				outs := make([]*types.Var, results)
				for i := range outs {
					outs[i] = types.NewParam(token.NoPos, nil, "", types.Typ[types.Uintptr])
				}
				sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(param), types.NewTuple(outs...), false)
				native := types.NewNamed(types.NewTypeName(token.NoPos, types.NewPackage(pkg.Path(), "p"), "Callback", nil), sig, nil)
				prog.SetTypeCallConv(pkg.Path()+".Callback", CDecl)
				source := pkg.NewFunc(pkg.Path()+".source", sig, GoDecl)
				body := source.MakeBody(1)
				values := make([]Expr, results)
				for i := range values {
					values[i] = source.Param(0)
				}
				body.Return(values...)
				owner := pkg.NewFunc(pkg.Path()+".owner", NoArgsNoRet, GoDecl)
				body = owner.MakeBody(1)
				dst := prog.Type(native, GoDecl)
				callback := body.ChangeType(dst, source.Expr)
				again := body.ChangeType(dst, source.Expr)
				if callback.impl != again.impl {
					t.Fatal("repeated conversion did not reuse the adapter")
				}
				goValue := checkExpr(source.Expr, prog.Type(sig, GoDecl).raw.Type, body)
				if fromValue := body.ChangeType(dst, goValue); fromValue.impl != callback.impl {
					t.Fatal("known Go funcval did not reuse the direct-entry adapter")
				}
				if nilCallback := body.ChangeType(dst, prog.Nil(prog.Type(sig, GoDecl))); !nilCallback.impl.IsNull() {
					t.Fatalf("nil conversion = %s", nilCallback.impl)
				}
				body.Return()
				if profile == "j64" {
					if callback.impl != source.impl {
						t.Fatal("equal Go/native prototypes should keep their entry")
					}
				} else {
					if callback.impl == source.impl || callback.impl.Linkage() != llvm.InternalLinkage {
						t.Fatal("Memory32 callback requires an internal native adapter")
					}
					if got := callback.impl.GlobalValueType().ParamTypes()[0].IntTypeWidth(); got != 32 {
						t.Fatalf("native parameter width = %d, want 32", got)
					}
					if !strings.Contains(callback.impl.String(), "call") || !strings.Contains(callback.impl.String(), "zext i32") ||
						results > 0 && !strings.Contains(callback.impl.String(), "trunc i64") {
						t.Fatalf("adapter did not convert Go/native words:\n%s", callback.impl)
					}
				}
				if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Fatalf("invalid callback adapter: %v\n%s", err, pkg.String())
				}
			})
		}
	}
}

func TestNativeUintptrFuncvalAdapter(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32"})
	defer prog.Dispose()
	setTestRuntime(t, prog)
	pkg := prog.NewPackage("p", "example.com/p")
	word := types.NewParam(token.NoPos, nil, "", types.Typ[types.Uintptr])
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(word), types.NewTuple(word), false)
	native := pkg.NewFunc("native", sig, CDecl)
	owner := pkg.NewFunc(pkg.Path()+".owner", NoArgsNoRet, GoDecl)
	body := owner.MakeBody(1)
	converted := body.ChangeType(prog.Type(sig, GoDecl), native.Expr)
	body.Return()
	entry := body.Field(converted, 0).impl
	if entry == native.impl || !strings.Contains(entry.String(), "call i32") || !strings.Contains(entry.String(), "zext i32") {
		t.Fatalf("native-to-Go conversion did not adapt uintptr:\n%s", pkg.String())
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestNativeUintptrCallbackRejectsDynamicGoFuncval(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32"})
	defer prog.Dispose()
	pkg := prog.NewPackage("p", "example.com/p")
	word := types.NewParam(token.NoPos, nil, "", types.Typ[types.Uintptr])
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(word), types.NewTuple(word), false)
	native := types.NewNamed(types.NewTypeName(token.NoPos, types.NewPackage(pkg.Path(), "p"), "Callback", nil), sig, nil)
	prog.SetTypeCallConv(pkg.Path()+".Callback", CDecl)
	ownerSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewParam(token.NoPos, nil, "fn", sig)), nil, false)
	owner := pkg.NewFunc(pkg.Path()+".owner", ownerSig, GoDecl)
	body := owner.MakeBody(1)
	defer func() {
		if got := recover(); got == nil || !strings.Contains(fmt.Sprint(got), nativeCallbackABIError) {
			t.Fatalf("dynamic callback diagnostic = %v", got)
		}
	}()
	body.ChangeType(prog.Type(native, GoDecl), owner.Param(0))
}
