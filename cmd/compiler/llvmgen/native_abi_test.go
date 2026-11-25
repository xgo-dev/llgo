package llvmgen

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llvm"
)

// Attribute slots are the result, int8, uint8, int16, uint16, bool, int32,
// and uint32. RISC-V64 sign-extends both signed and unsigned 32-bit arguments.
var nativeIntegerABITests = func() []struct {
	target Target
	attrs  []string
} {
	narrow := []string{"", "signext", "zeroext", "signext", "zeroext", "zeroext", "", ""}
	riscv64 := []string{"", "signext", "zeroext", "signext", "zeroext", "zeroext", "signext", "signext"}
	windows64 := []string{"", "", "", "", "", "zeroext", "", ""}
	return []struct {
		target Target
		attrs  []string
	}{
		{Target{GOOS: "darwin", GOARCH: "arm64"}, narrow},
		{Target{GOOS: "darwin", GOARCH: "amd64"}, narrow},
		{Target{GOOS: "linux", GOARCH: "amd64"}, narrow},
		{Target{GOOS: "linux", GOARCH: "386"}, narrow},
		{Target{GOOS: "linux", GOARCH: "arm", GOARM: "7"}, narrow},
		{Target{GOOS: "linux", GOARCH: "arm64"}, nil},
		{Target{GOOS: "linux", GOARCH: "arm", Target: "cortex-m3", LLVMTarget: "thumbv7m-unknown-unknown-eabi"}, narrow},
		{Target{GOOS: "linux", GOARCH: "arm", Target: "riscv32", LLVMTarget: "riscv32-unknown-elf"}, narrow},
		{Target{GOOS: "linux", GOARCH: "arm", Target: "riscv64", LLVMTarget: "riscv64-unknown-elf"}, riscv64},
		{Target{GOOS: "linux", GOARCH: "arm64", Target: "riscv64", LLVMTarget: "riscv64-unknown-elf"}, riscv64},
		{Target{GOOS: "linux", GOARCH: "arm", Target: "custom", LLVMTarget: "aarch64-unknown-linux-gnu"}, nil},
		{Target{GOOS: "windows", GOARCH: "amd64"}, windows64},
		{Target{GOOS: "windows", GOARCH: "amd64", LLVMTarget: "x86_64-w64-windows-gnu"}, windows64},
		{Target{GOOS: "windows", GOARCH: "386"}, narrow},
		{Target{GOOS: "windows", GOARCH: "386", LLVMTarget: "i686-w64-windows-gnu"}, narrow},
		{Target{GOOS: "windows", GOARCH: "arm64"}, nil},
		{Target{GOOS: "windows", GOARCH: "arm64", LLVMTarget: "aarch64-w64-windows-gnu"}, nil},
		{Target{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"}, narrow},
		{Target{GOOS: "js", GOARCH: "wasm"}, narrow},
	}
}()

func TestNativeNarrowIntegerAttributes(t *testing.T) {
	for _, test := range nativeIntegerABITests {
		target := test.target
		t.Run(target.GOOS+"/"+target.GOARCH+"/"+target.LLVMTarget, func(t *testing.T) {
			prog := NewProgram(&target)
			defer prog.Dispose()
			pkg := prog.NewPackage("p", "example.com/p")
			rawPkg := types.NewPackage(pkg.Path(), "p")
			signed := types.NewNamed(types.NewTypeName(token.NoPos, rawPkg, "Signed", nil), types.Typ[types.Int8], nil)
			unsigned := types.NewAlias(types.NewTypeName(token.NoPos, rawPkg, "Unsigned", nil), types.Typ[types.Uint16])
			paramTypes := []types.Type{signed, types.Typ[types.Uint8], types.Typ[types.Int16], unsigned, types.Typ[types.Bool], types.Typ[types.Int32], types.Typ[types.Uint32]}
			params := make([]*types.Var, len(paramTypes))
			for i, typ := range paramTypes {
				params[i] = types.NewVar(token.NoPos, nil, "", typ)
			}
			want := test.attrs
			if want == nil {
				want = make([]string, len(params)+1)
			}
			for _, resultIndex := range []int{0, 3, 4, 6} {
				sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), types.NewTuple(params[resultIndex]), false)
				name := types.TypeString(paramTypes[resultIndex], nil)
				native := pkg.NewFunc("native."+name, sig, CDecl)
				native.MakeBody(1).Return(native.Param(resultIndex))
				checkNarrowAttrs(t, native.impl, false, want)

				caller := pkg.NewFunc("direct."+name, sig, GoDecl)
				body := caller.MakeBody(1)
				args := make([]Expr, len(params))
				for i := range args {
					args[i] = caller.Param(i)
				}
				call := body.Call(native.Expr, args...)
				checkNarrowAttrs(t, call.impl, true, want)
				checkNarrowAttrs(t, caller.impl, false, make([]string, len(want)))
				if target.GOOS == "windows" {
					stdcall := pkg.NewFunc("stdcall."+name, sig, Stdcall)
					stdcall.MakeBody(1).Return(stdcall.Param(resultIndex))
					checkNarrowAttrs(t, stdcall.impl, false, want)
					checkNarrowAttrs(t, body.Call(stdcall.Expr, args...).impl, true, want)
				}
				body.Return(call)

				pointerName := "Pointer" + name
				pointer := types.NewNamed(types.NewTypeName(token.NoPos, rawPkg, pointerName, nil), sig, nil)
				prog.SetTypeCallConv(pkg.Path()+"."+pointerName, CDecl)
				pointerParam := types.NewVar(token.NoPos, nil, "fn", pointer)
				indirectSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(append([]*types.Var{pointerParam}, params...)...), sig.Results(), false)
				indirect := pkg.NewFunc("indirect."+name, indirectSig, GoDecl)
				body = indirect.MakeBody(1)
				for i := range args {
					args[i] = indirect.Param(i + 1)
				}
				call = body.Call(indirect.Param(0), args...)
				body.Return(call)
				checkNarrowAttrs(t, call.impl, true, want)
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatalf("invalid native integer ABI: %v\n%s", err, pkg.String())
			}
		})
	}
}

func TestNativeNarrowVariadicAttributes(t *testing.T) {
	for _, test := range []struct {
		target Target
		attrs  []string
	}{
		{Target{GOOS: "darwin", GOARCH: "arm64"}, []string{"", "signext", "", "", ""}},
		{Target{GOOS: "linux", GOARCH: "arm", Target: "riscv64", LLVMTarget: "riscv64-unknown-elf"}, []string{"", "signext", "signext", "signext", ""}},
	} {
		t.Run(test.target.GOOS+"/"+test.target.GOARCH+"/"+test.target.LLVMTarget, func(t *testing.T) {
			prog := NewProgram(&test.target)
			defer prog.Dispose()
			pkg := prog.NewPackage("p", "example.com/p")
			params := types.NewTuple(types.NewVar(token.NoPos, nil, "fixed", types.Typ[types.Int8]), VArg())
			sig := types.NewSignatureType(nil, nil, nil, params, nil, true)
			fn := pkg.NewFunc("native.variadic", sig, CDecl)
			checkNarrowAttrs(t, fn.impl, false, []string{"", "signext"})
			pointer := types.NewNamed(types.NewTypeName(token.NoPos, types.NewPackage(pkg.Path(), "p"), "Native", nil), sig, nil)
			prog.SetTypeCallConv(pkg.Path()+".Native", CDecl)
			callerSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "fn", pointer)), nil, false)
			caller := pkg.NewFunc("caller", callerSig, GoDecl)
			body := caller.MakeBody(1)
			args := []Expr{
				prog.IntVal(248, prog.Type(types.Typ[types.Int8], GoDecl)),
				prog.IntVal(42, prog.Int32()),
				prog.IntVal(0xffffffff, prog.Type(types.Typ[types.Uint32], GoDecl)),
				prog.IntVal(42, prog.Int64()),
			}
			checkNarrowAttrs(t, body.Call(fn.Expr, args...).impl, true, test.attrs)
			checkNarrowAttrs(t, body.Call(caller.Param(0), args...).impl, true, test.attrs)
			body.Return()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func checkNarrowAttrs(t *testing.T, value llvm.Value, call bool, want []string) {
	t.Helper()
	for i, expected := range want {
		for _, name := range []string{"signext", "zeroext"} {
			kind := llvm.AttributeKindID(name)
			var attr llvm.Attribute
			if call {
				attr = value.GetCallSiteEnumAttribute(i, kind)
			} else {
				attr = value.GetEnumAttributeAtIndex(i, kind)
			}
			if !attr.IsNil() != (expected == name) {
				t.Errorf("attribute %s at index %d: want %q\n%s", name, i, expected, value.String())
			}
		}
	}
}
