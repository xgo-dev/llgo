package llvmgen

import (
	"github.com/xgo-dev/llgo/types"
	"go/token"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestStringABIWithoutPointerPadding(t *testing.T) {
	for _, target := range []Target{
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm64-unknown-emscripten", WasmProfile: "j64"},
		{GOOS: "wasip1", GOARCH: "wasm"},
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "windows", GOARCH: "amd64"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			prog := NewProgram(&target)
			defer prog.Dispose()
			setTestRuntime(t, prog)
			pkg := prog.NewPackage("p", "example.com/p")
			text := types.NewTuple(types.NewParam(token.NoPos, nil, "", types.Typ[types.String]))
			sig := types.NewSignatureType(nil, nil, nil, text, text, false)
			native := pkg.NewFunc("native", sig, CDecl)
			caller := pkg.NewFunc("caller", sig, GoDecl)
			arg := caller.Param(0)
			b := caller.MakeBody(1)
			defer b.Dispose()
			fn := b.ChangeType(prog.Type(sig, GoDecl), native.Expr)
			if b.Field(fn, 0).impl != native.impl {
				t.Fatal("string-only signature acquired a funcval adapter")
			}
			result := b.Call(native.Expr, arg)
			b.Return(result)
			b.EndBuild()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			call := result.impl
			if call.IsACallInst().IsNil() || call.CalledValue() != native.impl || call.Operand(0) != caller.impl.Param(0) {
				t.Fatalf("expected the native call to forward its string argument and result directly:\n%s", caller.impl)
			}
		})
	}
}

func TestWasmStringParamBeforeBody(t *testing.T) {
	for _, cc := range []struct {
		name string
		conv CallConv
	}{
		{"go", GoDecl},
		{"native", CDecl},
	} {
		testWasmStringProfiles(t, cc.name, func(t *testing.T, prog Program, pkg Package) {
			text := types.NewParam(token.NoPos, nil, "text", types.Typ[types.String])
			which := types.NewParam(token.NoPos, nil, "which", types.Typ[types.Bool])
			result := types.NewParam(token.NoPos, nil, "", types.Typ[types.Uint32])
			sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(text, which), types.NewTuple(result), false)

			t.Run("declaration", func(t *testing.T) {
				fn := pkg.NewFunc(pkg.Path()+".external", sig, cc.conv)
				_ = fn.Param(0)
				if fn.HasBody() || !fn.impl.IsDeclaration() {
					t.Fatal("retrieving a parameter defined an external function")
				}
			})

			t.Run("unused", func(t *testing.T) {
				fn := pkg.NewFunc(pkg.Path()+".unused", sig, cc.conv)
				_ = fn.Param(0)
				body := fn.MakeBody(1)
				defer body.Dispose()
				body.Return(prog.IntVal(0, prog.Uint32()))
				body.EndBuild()
				if !fn.impl.Param(0).FirstUse().IsNil() {
					t.Fatalf("unused string parameter generated instructions:\n%s", fn.impl)
				}
			})

			t.Run("multiple-blocks", func(t *testing.T) {
				fn := pkg.NewFunc(pkg.Path()+".early", sig, cc.conv)
				// Clients may retain a parameter before they create the body.
				// Its uses must still see a canonical pointer in every block.
				value := fn.Param(0)
				body := fn.MakeBody(3)
				defer body.Dispose()
				body.If(fn.Param(1), fn.Block(1), fn.Block(2))
				for i, param := range []Expr{value, fn.Param(0)} {
					body.SetBlock(fn.Block(i + 1))
					body.Return(nativeStringHighWords(body, param, [][]int{nil}))
				}
				body.EndBuild()
				checkNativeStringZeroResults(t, pkg, fn.impl)
			})
		})
	}
}
