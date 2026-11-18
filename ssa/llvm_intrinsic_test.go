package ssa

import (
	"go/importer"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestLLVMIntrinsicConstantOperands(t *testing.T) {
	for _, tc := range []struct {
		name   string
		params []types.Type
		want   []int
	}{
		{"llvm.memcpy", []types.Type{types.Typ[types.UnsafePointer], types.Typ[types.UnsafePointer], types.Typ[types.Uintptr], types.Typ[types.Bool]}, []int{3}},
		{"llvm.memcpy.inline", []types.Type{types.Typ[types.UnsafePointer], types.Typ[types.UnsafePointer], types.Typ[types.Uintptr], types.Typ[types.Bool]}, []int{2, 3}},
		{"llvm.memmove", []types.Type{types.Typ[types.UnsafePointer], types.Typ[types.UnsafePointer], types.Typ[types.Uintptr], types.Typ[types.Bool]}, []int{3}},
		{"llvm.memset", []types.Type{types.Typ[types.UnsafePointer], types.Typ[types.Byte], types.Typ[types.Uintptr], types.Typ[types.Bool]}, []int{3}},
		{"llvm.memset.inline", []types.Type{types.Typ[types.UnsafePointer], types.Typ[types.Byte], types.Typ[types.Uintptr], types.Typ[types.Bool]}, []int{2, 3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := intrinsicDeclarationPackage(t, nil)
			params := make([]*types.Var, len(tc.params))
			for i, typ := range tc.params {
				params[i] = types.NewParam(0, nil, "", typ)
			}
			sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), nil, false)
			d, err := pkg.Prog.ResolveLLVMIntrinsic(tc.name, sig)
			if err != nil {
				t.Fatal(err)
			}
			fn := pkg.NewFunc("caller", sig, CDecl)
			b := fn.MakeBody(1)
			defer b.Dispose()
			args := make([]Expr, len(tc.params))
			for i := range args {
				args[i] = fn.Param(i)
			}
			for _, i := range tc.want {
				if tc.params[i] == types.Typ[types.Bool] {
					args[i] = pkg.Prog.Val(false)
				} else {
					args[i] = pkg.Prog.IntVal(8, args[i].Type)
				}
			}
			for _, i := range tc.want {
				valid := args[i]
				args[i] = fn.Param(i)
				if _, err := b.CallLLVMIntrinsic(d, args...); err == nil || !strings.Contains(err.Error(), "immediate constant") {
					t.Fatalf("dynamic operand %d: %v", i, err)
				}
				args[i] = valid
			}
			if _, err := b.CallLLVMIntrinsic(d, args...); err != nil {
				t.Fatal(err)
			}
			b.Return()
			b.EndBuild()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLLVMIntrinsicCallRejectsInvalidOperandsAndDeclarations(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"missing-argument", "expected 1 arguments, got 0"},
		{"extra-argument", "expected 1 arguments, got 2"},
		{"argument-type", "argument 1 has type i16, expected i32"},
		{"declaration-type", "conflicting LLVM intrinsic declaration"},
		{"calling-convention", "conflicting LLVM intrinsic declaration"},
		{"definition", "conflicting LLVM intrinsic declaration"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := intrinsicDeclarationPackage(t, nil)
			prog := pkg.Prog
			param := types.NewParam(0, nil, "x", types.Typ[types.Uint32])
			sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(param), types.NewTuple(param), false)
			d, err := prog.ResolveLLVMIntrinsic("llvm.ctpop", sig)
			if err != nil {
				t.Fatal(err)
			}
			fn := pkg.NewFunc("caller", sig, CDecl)
			b := fn.MakeBody(1)
			defer b.Dispose()
			args := []Expr{fn.Param(0)}
			declType := d.binding.Signature()
			declName := d.binding.Name()
			var conflict llvm.Value
			switch tc.name {
			case "missing-argument":
				args = nil
			case "extra-argument":
				args = append(args, args[0])
			case "argument-type":
				args[0] = prog.IntVal(1, prog.Uint16())
			case "declaration-type":
				conflict = llvm.AddFunction(pkg.Module(), declName, llvm.FunctionType(prog.Uint16().ll, declType.ParamTypes(), false))
			case "calling-convention", "definition":
				conflict = llvm.AddFunction(pkg.Module(), declName, declType)
				if tc.name == "calling-convention" {
					conflict.SetFunctionCallConv(llvm.FastCallConv)
				} else {
					body := prog.ctx.NewBuilder()
					body.SetInsertPointAtEnd(prog.ctx.AddBasicBlock(conflict, "entry"))
					body.CreateRet(llvm.ConstNull(declType.ReturnType()))
					body.Dispose()
				}
			}
			ret, err := b.CallLLVMIntrinsic(d, args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) || !ret.IsNil() {
				t.Fatalf("invalid call = (%v, %v), want nil result and %q", ret, err, tc.want)
			}
			if !conflict.IsNil() {
				conflict.EraseFromParentAsFunction()
			}
			// Rejection must leave the builder usable and must not poison the
			// checked declaration or subsequent intrinsic lookup.
			ret, err = b.CallLLVMIntrinsic(d, fn.Param(0))
			if err != nil {
				t.Fatal(err)
			}
			b.Return(ret)
			b.EndBuild()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLLVMIntrinsicGoSignatureMapping(t *testing.T) {
	pkg := intrinsicDeclarationPackage(t, nil)
	pkg.Prog.SetRuntime(func() *types.Package {
		rt, err := importer.For("source", nil).Import(PkgRuntime)
		if err != nil {
			t.Fatal(err)
		}
		return rt
	})
	anyType := types.NewInterfaceType(nil, nil).Complete()
	fixed := types.NewParam(0, nil, "fixed", types.Typ[types.Uint32])
	for _, tc := range []struct {
		name, marker         string
		elem                 types.Type
		variadic, wantVararg bool
		wantParams           int
		wantError            string
	}{
		{"marked", NameValist, anyType, true, true, 1, ""},
		{"ordinary", "args", anyType, true, false, 2, ""},
		{"typed-marker", NameValist, types.Typ[types.Uint32], true, true, 0, "must have type ...any"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			last := types.NewParam(0, nil, tc.marker, types.NewSlice(tc.elem))
			sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(fixed, last), nil, tc.variadic)
			ft, _, err := pkg.Prog.llvmIntrinsicSignature(sig)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if ft.IsFunctionVarArg() != tc.wantVararg || ft.ParamTypesCount() != tc.wantParams {
				t.Fatalf("incorrect LLVM signature: %s", ft)
			}
			if !tc.wantVararg && ft.ParamTypes()[1].TypeKind() != llvm.StructTypeKind {
				t.Fatal("ordinary Go variadic slice was flattened")
			}
		})
	}
	array := types.NewArray(types.Typ[types.Uint8], 3)
	arg := types.NewParam(0, nil, "x", array)
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(arg), types.NewTuple(arg), false)
	ft, _, err := pkg.Prog.llvmIntrinsicSignature(sig)
	if err != nil {
		t.Fatal(err)
	}
	if ft.ReturnType().TypeKind() != llvm.ArrayTypeKind || ft.ParamTypes()[0] != ft.ReturnType() {
		t.Fatalf("array changed representation: %s", ft)
	}
}
