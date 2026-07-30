package wasmresume

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestLowerMovesPersistentDynamicAllocaIntoContextStorage(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("dynamic-alloca")
	defer mod.Dispose()
	mod.SetTarget("wasm32-unknown-unknown")
	targetData := llvm.NewTargetData("e-m:e-p:32:32-i64:64-n32:64-S128")
	defer targetData.Dispose()

	i8 := ctx.Int8Type()
	ptr := llvm.PointerType(i8, 0)
	calleeType := llvm.FunctionType(ctx.VoidType(), []llvm.Type{ptr}, false)
	callee := llvm.AddFunction(mod, "callee", calleeType)
	markFunction(ctx, callee)
	calleeBlock := ctx.AddBasicBlock(callee, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(calleeBlock)
	builder.CreateRetVoid()

	caller := llvm.AddFunction(mod, "caller", llvm.FunctionType(ctx.VoidType(), []llvm.Type{ctx.Int32Type()}, false))
	markFunction(ctx, caller)
	block := ctx.AddBasicBlock(caller, "entry")
	builder.SetInsertPointAtEnd(block)
	buffer := builder.CreateArrayAlloca(i8, caller.Param(0), "buffer")
	call := builder.CreateCall(calleeType, callee, []llvm.Value{buffer}, "")
	markCall(ctx, call)
	builder.CreateRetVoid()

	if err := Lower(mod, targetData); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify lowered module: %v\n%s", err, mod.String())
	}
	ir := mod.String()
	if strings.Contains(ir, "%buffer = alloca") ||
		!strings.Contains(ir, "call ptr @__llgo_wasm_resume_alloc_dynamic") {
		t.Fatalf("dynamic alloca was not moved into context storage:\n%s", ir)
	}
}

func TestLowerRemovesStackLifetimeAcrossResume(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("stack-lifetime")
	defer mod.Dispose()
	mod.SetTarget("wasm32-unknown-unknown")
	targetData := llvm.NewTargetData("e-m:e-p:32:32-i64:64-n32:64-S128")
	defer targetData.Dispose()

	callee := llvm.AddFunction(mod, "callee", llvm.FunctionType(ctx.VoidType(), nil, false))
	markFunction(ctx, callee)
	calleeBlock := ctx.AddBasicBlock(callee, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(calleeBlock)
	builder.CreateRetVoid()

	caller := llvm.AddFunction(mod, "caller", llvm.FunctionType(ctx.VoidType(), nil, false))
	markFunction(ctx, caller)
	block := ctx.AddBasicBlock(caller, "entry")
	builder.SetInsertPointAtEnd(block)
	saved := builder.CreateIntrinsic(
		llvm.PointerType(ctx.Int8Type(), 0),
		llvm.LookupIntrinsicID("llvm.stacksave"),
		nil,
		"",
	)
	call := builder.CreateCall(callee.GlobalValueType(), callee, nil, "")
	markCall(ctx, call)
	builder.CreateIntrinsic(
		ctx.VoidType(),
		llvm.LookupIntrinsicID("llvm.stackrestore"),
		[]llvm.Value{saved},
		"",
	)
	builder.CreateRetVoid()

	if err := Lower(mod, targetData); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify lowered module: %v\n%s", err, mod.String())
	}
	if ir := mod.String(); strings.Contains(ir, "call ptr @llvm.stacksave") ||
		strings.Contains(ir, "call void @llvm.stackrestore") {
		t.Fatalf("native stack lifetime crosses a resume point:\n%s", ir)
	}
}
