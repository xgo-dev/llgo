package ssa

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestNativeNarrowIntegerAttributes(t *testing.T) {
	for _, target := range []Target{
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "darwin", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "amd64", LLVMTarget: "x86_64-w64-windows-gnu"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "windows", GOARCH: "386", LLVMTarget: "i686-w64-windows-gnu"},
		{GOOS: "windows", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "arm64", LLVMTarget: "aarch64-w64-windows-gnu"},
		{GOOS: "wasip1", GOARCH: "wasm"},
		{GOOS: "js", GOARCH: "wasm"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH+"/"+target.LLVMTarget, func(t *testing.T) {
			prog := NewProgram(&target)
			defer prog.Dispose()
			pkg := prog.NewPackage("p", "example.com/p")
			rawPkg := types.NewPackage(pkg.Path(), "p")
			signed := types.NewNamed(types.NewTypeName(token.NoPos, rawPkg, "Signed", nil), types.Typ[types.Int8], nil)
			unsigned := types.NewAlias(types.NewTypeName(token.NoPos, rawPkg, "Unsigned", nil), types.Typ[types.Uint16])
			paramTypes := []types.Type{signed, types.Typ[types.Uint8], types.Typ[types.Int16], unsigned, types.Typ[types.Bool], types.Typ[types.Int32]}
			params := make([]*types.Var, len(paramTypes))
			for i, typ := range paramTypes {
				params[i] = types.NewVar(token.NoPos, nil, "", typ)
			}
			want := []string{"", "signext", "zeroext", "signext", "zeroext", "zeroext", ""}
			if target.GOARCH == "arm64" && target.GOOS != "darwin" {
				want = make([]string, len(want))
			} else if target.GOARCH == "amd64" && target.GOOS == "windows" {
				want = []string{"", "", "", "", "", "zeroext", ""}
			}
			for _, resultIndex := range []int{0, 3, 4} {
				sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), types.NewTuple(params[resultIndex]), false)
				name := types.TypeString(paramTypes[resultIndex], nil)
				native := pkg.NewFunc("native."+name, sig, InC)
				native.MakeBody(1).Return(native.Param(resultIndex))
				checkNarrowAttrs(t, native.impl, false, want)

				caller := pkg.NewFunc("direct."+name, sig, InGo)
				body := caller.MakeBody(1)
				args := make([]Expr, len(params))
				for i := range args {
					args[i] = caller.Param(i)
				}
				call := body.Call(native.Expr, args...)
				checkNarrowAttrs(t, call.impl, true, want)
				checkNarrowAttrs(t, caller.impl, false, make([]string, len(want)))
				if target.GOOS == "windows" {
					stdcall := pkg.NewFunc("stdcall."+name, sig, InStdcall)
					stdcall.MakeBody(1).Return(stdcall.Param(resultIndex))
					checkNarrowAttrs(t, stdcall.impl, false, want)
					checkNarrowAttrs(t, body.Call(stdcall.Expr, args...).impl, true, want)
				}
				body.Return(call)

				pointerName := "Pointer" + name
				pointer := types.NewNamed(types.NewTypeName(token.NoPos, rawPkg, pointerName, nil), sig, nil)
				prog.SetTypeBackground(pkg.Path()+"."+pointerName, InC)
				pointerParam := types.NewVar(token.NoPos, nil, "fn", pointer)
				indirectSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(append([]*types.Var{pointerParam}, params...)...), sig.Results(), false)
				indirect := pkg.NewFunc("indirect."+name, indirectSig, InGo)
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
	prog := NewProgram(&Target{GOOS: "darwin", GOARCH: "arm64"})
	defer prog.Dispose()
	pkg := prog.NewPackage("p", "example.com/p")
	params := types.NewTuple(
		types.NewVar(token.NoPos, nil, "fixed", types.Typ[types.Int8]),
		types.NewVar(token.NoPos, nil, "__llgo_va_list", types.NewSlice(types.NewInterfaceType(nil, nil).Complete())),
	)
	sig := types.NewSignatureType(nil, nil, nil, params, nil, true)
	fn := pkg.NewFunc("native.variadic", sig, InC)
	checkNarrowAttrs(t, fn.impl, false, []string{"", "signext"})
	caller := pkg.NewFunc("caller", NoArgsNoRet, InGo)
	body := caller.MakeBody(1)
	call := body.Call(fn.Expr, prog.IntVal(248, prog.Type(types.Typ[types.Int8], InGo)), prog.IntVal(42, prog.Int32()))
	body.Return()
	checkNarrowAttrs(t, call.impl, true, []string{"", "signext", ""})
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
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
