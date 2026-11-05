package ssa

import (
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

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
			var conflict llvm.Value
			switch tc.name {
			case "missing-argument":
				args = nil
			case "extra-argument":
				args = append(args, args[0])
			case "argument-type":
				args[0] = prog.IntVal(1, prog.Uint16())
			case "declaration-type":
				conflict = llvm.AddFunction(pkg.Module(), d.name, llvm.FunctionType(prog.Uint16().ll, d.params, false))
			case "calling-convention", "definition":
				conflict = llvm.AddFunction(pkg.Module(), d.name, llvm.FunctionType(d.ret, d.params, false))
				if tc.name == "calling-convention" {
					conflict.SetFunctionCallConv(llvm.FastCallConv)
				} else {
					body := prog.ctx.NewBuilder()
					body.SetInsertPointAtEnd(prog.ctx.AddBasicBlock(conflict, "entry"))
					body.CreateRet(llvm.ConstNull(d.ret))
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
