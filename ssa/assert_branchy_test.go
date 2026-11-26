package ssa

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

// assertBranchyShape checks the branchy lowering shared by the runtime assert
// guards: the entry block ends in a conditional branch carrying the original
// condition and contains no call, the cold block opens with the named runtime
// call passing constant i1 true, and the cold block terminates per the
// frame-pointer target policy (formal return vs self-loop). It returns the
// success block so callers can keep asserting the hot-path code.
func assertBranchyShape(t *testing.T, prog Program, fn Function, callee string, fnImpl llvm.Value) llvm.BasicBlock {
	t.Helper()
	entry := fnImpl.EntryBasicBlock()
	branch := entry.LastInstruction()
	if branch.InstructionOpcode() != llvm.Br || branch.SuccessorsCount() != 2 {
		t.Fatalf("%s check must branch before calling runtime:\n%s", callee, fnImpl.String())
	}
	for inst := entry.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
		if inst.InstructionOpcode() == llvm.Call {
			t.Fatalf("%s check calls runtime unconditionally:\n%s", callee, fnImpl.String())
		}
	}
	failure := branch.Successor(0)
	first := failure.FirstInstruction()
	if first.InstructionOpcode() != llvm.Call {
		t.Fatalf("%s failure block must open with the runtime call:\n%s", callee, fnImpl.String())
	}
	called := first.CalledValue().IsAFunction()
	if called.IsNil() || called.Name() != PkgRuntime+"."+callee {
		t.Fatalf("unexpected %s failure callee:\n%s", callee, fnImpl.String())
	}
	arg0 := first.Operand(0)
	if arg0.IsAConstantInt().IsNil() || arg0.Type().String() != "i1" || arg0.ZExtValue() != 1 {
		t.Fatalf("%s failure must pass constant i1 true:\n%s", callee, fnImpl.String())
	}
	next := failure.LastInstruction()
	if prog.NeedsFramePointer() {
		if next.InstructionOpcode() != llvm.Ret {
			t.Fatalf("native %s failure must retain a separate formal return:\n%s", callee, fnImpl.String())
		}
	} else if next.InstructionOpcode() != llvm.Br || next.SuccessorsCount() != 1 || next.Successor(0) != failure {
		t.Fatalf("non-native %s failure must not rejoin the success path:\n%s", callee, fnImpl.String())
	}
	return branch.Successor(1)
}

func TestRuntimeAssertsBranchy(t *testing.T) {
	Initialize(InitAllTargets | InitAllTargetInfos | InitAllTargetMCs | InitAllAsmPrinters)
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "amd64", LLVMTarget: "x86_64-unknown-linux"},
		{GOOS: "darwin", GOARCH: "arm64", LLVMTarget: "aarch64-apple-darwin"},
		{GOOS: "windows", GOARCH: "amd64", LLVMTarget: "x86_64-pc-windows-msvc"},
		{GOOS: "windows", GOARCH: "386", LLVMTarget: "i686-pc-windows-msvc"},
		{GOOS: "windows", GOARCH: "arm64", LLVMTarget: "aarch64-pc-windows-msvc"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-emscripten"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm64-unknown-emscripten"},
		{GOOS: "wasip1", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-wasip1"},
	} {
		t.Run(target.LLVMTarget, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			prog.SetRuntime(func() *types.Package {
				rt := types.NewPackage(PkgRuntime, "runtime")
				boolOnly := func(name string) {
					params := types.NewTuple(types.NewVar(token.NoPos, nil, "b", types.Typ[types.Bool]))
					rt.Scope().Insert(types.NewFunc(token.NoPos, rt, name, types.NewSignatureType(nil, nil, nil, params, nil, false)))
				}
				boolOnly("AssertDivideByZero")
				boolOnly("AssertNegativeShift")
				// The message descriptor mirrors runtime.String ({data, len}) so
				// b.Str can build it without importing the runtime package; the
				// real string spelling is covered by the IR fixtures.
				str := types.NewStruct([]*types.Var{
					types.NewField(token.NoPos, rt, "data", types.Typ[types.UnsafePointer], false),
					types.NewField(token.NoPos, rt, "len", types.Typ[types.Int], false),
				}, nil)
				strName := types.NewTypeName(token.NoPos, rt, "String", nil)
				types.NewNamed(strName, str, nil)
				rt.Scope().Insert(strName)
				params := types.NewTuple(
					types.NewVar(token.NoPos, nil, "b", types.Typ[types.Bool]),
					types.NewVar(token.NoPos, nil, "msg", types.Typ[types.String]),
				)
				rt.Scope().Insert(types.NewFunc(token.NoPos, rt, "AssertRuntimeError", types.NewSignatureType(nil, nil, nil, params, nil, false)))
				return rt
			})
			pkg := prog.NewPackage("p", "example.com/asserts")
			zero := llvm.ConstInt(prog.Int64().ll, 0, false)
			twoI64 := types.NewTuple(
				types.NewVar(token.NoPos, nil, "x", types.Typ[types.Int64]),
				types.NewVar(token.NoPos, nil, "y", types.Typ[types.Int64]),
			)
			oneI64 := types.NewTuple(types.NewVar(token.NoPos, nil, "x", types.Typ[types.Int64]))

			// Integer division: the guard condition (divisor == 0) stays on the
			// hot path and is reused by the zero-safe divisor select below it.
			div := pkg.NewFunc("div", types.NewSignatureType(nil, nil, nil, twoI64, nil, false), GoDecl)
			db := div.MakeBody(1)
			db.BinOp(token.QUO, div.Param(0), div.Param(1))
			db.Return()
			db.EndBuild()
			divIR := div.impl.String()
			entryBranch := div.impl.EntryBasicBlock().LastInstruction()
			cond := entryBranch.Operand(0)
			if cond.IsAInstruction().IsNil() || cond.InstructionOpcode() != llvm.ICmp {
				t.Fatalf("divide-by-zero branch must keep the divisor icmp on the hot path:\n%s", divIR)
			}
			success := assertBranchyShape(t, prog, div, "AssertDivideByZero", div.impl)
			sel := success.FirstInstruction()
			if sel.InstructionOpcode() != llvm.Select || sel.Operand(0) != cond {
				t.Fatalf("success block must reuse the guard condition for the safe divisor:\n%s", divIR)
			}

			// Shifts: a signed count keeps its negative-shift guard off the hot
			// path while the shift itself stays in the success block.
			shl := pkg.NewFunc("shl", types.NewSignatureType(nil, nil, nil, twoI64, nil, false), GoDecl)
			sb := shl.MakeBody(1)
			sb.BinOp(token.SHL, shl.Param(0), shl.Param(1))
			sb.Return()
			sb.EndBuild()
			cond = shl.impl.EntryBasicBlock().LastInstruction().Operand(0)
			if cond.IsAInstruction().IsNil() || cond.InstructionOpcode() != llvm.ICmp {
				t.Fatalf("negative-shift branch must keep the count icmp on the hot path:\n%s", shl.impl.String())
			}
			success = assertBranchyShape(t, prog, shl, "AssertNegativeShift", shl.impl)
			if success.FirstInstruction().InstructionOpcode() == llvm.Call {
				t.Fatalf("shift must resume in the success block:\n%s", shl.impl.String())
			}

			// Generic runtime errors: the cold call still receives the message
			// string it would have built inline.
			rterr := pkg.NewFunc("rterr", types.NewSignatureType(nil, nil, nil, oneI64, nil, false), GoDecl)
			rb := rterr.MakeBody(1)
			check := llvm.CreateICmp(rb.impl, llvm.IntEQ, rterr.Param(0).impl, zero)
			const msg = "unsafe.Slice: len out of range"
			rb.assertRuntimeError(check, msg)
			rb.Return()
			rb.EndBuild()
			entryBranch = rterr.impl.EntryBasicBlock().LastInstruction()
			if entryBranch.Operand(0) != check {
				t.Fatalf("runtime-error branch lost the guard condition:\n%s", rterr.impl.String())
			}
			success = assertBranchyShape(t, prog, rterr, "AssertRuntimeError", rterr.impl)
			if got := success.FirstInstruction().InstructionOpcode(); got == llvm.Call {
				t.Fatalf("runtime-error guard must resume in the success block:\n%s", rterr.impl.String())
			}
			if !strings.Contains(pkg.Module().String(), msg) {
				t.Fatalf("runtime-error message missing from module globals:\n%s", pkg.Module().String())
			}

			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
