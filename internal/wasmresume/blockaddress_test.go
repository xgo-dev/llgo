package wasmresume

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestLowerRemapsBlockAddressesToResumeEntry(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("block-address")
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
	entry := ctx.AddBasicBlock(caller, "entry")
	target := ctx.AddBasicBlock(caller, "target")
	builder.SetInsertPointAtEnd(entry)
	call := builder.CreateCall(callee.GlobalValueType(), callee, nil, "")
	markCall(ctx, call)
	indirect := builder.CreateIndirectBr(llvm.BlockAddress(caller, target), 1)
	indirect.AddDest(target)
	builder.SetInsertPointAtEnd(target)
	builder.CreateRetVoid()

	if err := Lower(mod, targetData); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify lowered module: %v\n%s", err, mod.String())
	}
	ir := mod.String()
	if strings.Contains(ir, "blockaddress(@caller,") ||
		!strings.Contains(ir, "blockaddress(@__llgo_wasm_resume.caller,") {
		t.Fatalf("block address was not remapped to the resume entry:\n%s", ir)
	}
}
