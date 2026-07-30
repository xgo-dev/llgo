package wasmresume

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestSplitBlockAfterMovesContinuationAndRewritesPhi(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("split")
	defer mod.Dispose()

	i1 := ctx.Int1Type()
	i32 := ctx.Int32Type()
	voidFn := llvm.FunctionType(ctx.VoidType(), nil, false)
	callee := llvm.AddFunction(mod, "callee", voidFn)
	fn := llvm.AddFunction(mod, "caller", llvm.FunctionType(i32, []llvm.Type{i1, i32}, false))
	entry := ctx.AddBasicBlock(fn, "entry")
	other := ctx.AddBasicBlock(fn, "other")
	merge := ctx.AddBasicBlock(fn, "merge")
	builder := ctx.NewBuilder()
	defer builder.Dispose()

	builder.SetInsertPointAtEnd(entry)
	call := builder.CreateCall(voidFn, callee, nil, "")
	value := builder.CreateAdd(fn.Param(1), llvm.ConstInt(i32, 2, false), "value")
	builder.CreateBr(merge)
	builder.SetInsertPointAtEnd(other)
	builder.CreateBr(merge)
	builder.SetInsertPointAtEnd(merge)
	phi := builder.CreatePHI(i32, "selected")
	phi.AddIncoming(
		[]llvm.Value{value, llvm.ConstInt(i32, 0, false)},
		[]llvm.BasicBlock{entry, other},
	)
	builder.CreateRet(phi)

	continuation, err := splitBlockAfter(ctx, call, "resume.1")
	if err != nil {
		t.Fatal(err)
	}
	if continuation != llvm.NextBasicBlock(entry) {
		t.Fatal("continuation was not placed after the split block")
	}
	if got := llvm.NextInstruction(call).InstructionOpcode(); got != llvm.Br {
		t.Fatalf("split block terminator = %v, want br", got)
	}
	if got := continuation.FirstInstruction().Name(); got != "value" {
		t.Fatalf("first continuation instruction = %q, want value:\n%s", got, mod.String())
	}
	nextPhi := merge.FirstInstruction()
	if nextPhi.InstructionOpcode() != llvm.PHI || nextPhi.Name() != "selected" {
		t.Fatalf("replacement phi = %v %q", nextPhi.InstructionOpcode(), nextPhi.Name())
	}
	if nextPhi.IncomingBlock(0) != continuation || nextPhi.IncomingBlock(1) != other {
		t.Fatal("replacement phi has incorrect predecessors")
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify split module: %v\n%s", err, mod.String())
	}
}

func TestSplitBlockAfterRejectsInvalidPoints(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("invalid-split")
	defer mod.Dispose()
	fn := llvm.AddFunction(mod, "function", llvm.FunctionType(ctx.VoidType(), nil, false))
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	ret := builder.CreateRetVoid()

	if _, err := splitBlockAfter(ctx, ret, "resume"); err == nil ||
		!strings.Contains(err.Error(), "not a call") {
		t.Fatalf("non-call split error = %v", err)
	}

	callBlock := ctx.AddBasicBlock(fn, "unterminated")
	builder.SetInsertPointAtEnd(callBlock)
	callee := llvm.AddFunction(mod, "callee", llvm.FunctionType(ctx.VoidType(), nil, false))
	call := builder.CreateCall(callee.GlobalValueType(), callee, nil, "")
	if _, err := splitBlockAfter(ctx, call, "resume"); err == nil ||
		!strings.Contains(err.Error(), "no continuation") {
		t.Fatalf("terminal call split error = %v", err)
	}
}
