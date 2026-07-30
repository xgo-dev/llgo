package wasmresume

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestSpillValueStoresDefinitionAndReloadsUses(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("spill")
	defer mod.Dispose()
	i32 := ctx.Int32Type()
	frameType := ctx.StructType([]llvm.Type{i32}, false)
	fn := llvm.AddFunction(mod, "function", llvm.FunctionType(i32, []llvm.Type{
		llvm.PointerType(frameType, 0), i32,
	}, false))
	fn.Param(1).SetName("input")
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	field := builder.CreateStructGEP(frameType, fn.Param(0), 0, "field")
	value := builder.CreateAdd(fn.Param(1), llvm.ConstInt(i32, 1, false), "value")
	result := builder.CreateMul(value, value, "result")
	builder.CreateRet(result)

	if err := spillValue(ctx, value, field); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify spilled module: %v\n%s", err, mod.String())
	}
	ir := mod.String()
	if !strings.Contains(ir, "store i32 %value, ptr %field") ||
		strings.Count(ir, "load i32, ptr %field") != 1 ||
		!strings.Contains(ir, "mul i32 %value.reload, %value.reload") {
		t.Fatalf("value was not canonicalized through the frame:\n%s", ir)
	}
}

func TestSpillValueReloadsParameter(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("spill-parameter")
	defer mod.Dispose()
	i32 := ctx.Int32Type()
	frameType := ctx.StructType([]llvm.Type{i32}, false)
	fn := llvm.AddFunction(mod, "function", llvm.FunctionType(i32, []llvm.Type{
		llvm.PointerType(frameType, 0), i32,
	}, false))
	fn.Param(1).SetName("input")
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	field := builder.CreateStructGEP(frameType, fn.Param(0), 0, "field")
	builder.CreateRet(fn.Param(1))

	if err := spillValue(ctx, fn.Param(1), field); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify parameter reload: %v\n%s", err, mod.String())
	}
	if !strings.Contains(mod.String(), "ret i32 %input.reload") {
		t.Fatalf("parameter use was not loaded from the frame:\n%s", mod.String())
	}
}

func TestSpillValueStoresPhiAfterPhiGroup(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("spill-phi-definition")
	defer mod.Dispose()
	i1 := ctx.Int1Type()
	i32 := ctx.Int32Type()
	frameType := ctx.StructType([]llvm.Type{i32}, false)
	fn := llvm.AddFunction(mod, "function", llvm.FunctionType(i32, []llvm.Type{
		llvm.PointerType(frameType, 0), i1, i32,
	}, false))
	entry := ctx.AddBasicBlock(fn, "entry")
	left := ctx.AddBasicBlock(fn, "left")
	right := ctx.AddBasicBlock(fn, "right")
	merge := ctx.AddBasicBlock(fn, "merge")
	builder := ctx.NewBuilder()
	defer builder.Dispose()

	builder.SetInsertPointAtEnd(entry)
	field := builder.CreateStructGEP(frameType, fn.Param(0), 0, "field")
	builder.CreateCondBr(fn.Param(1), left, right)
	builder.SetInsertPointAtEnd(left)
	builder.CreateBr(merge)
	builder.SetInsertPointAtEnd(right)
	builder.CreateBr(merge)
	builder.SetInsertPointAtEnd(merge)
	phi := builder.CreatePHI(i32, "selected")
	phi.AddIncoming([]llvm.Value{
		fn.Param(2), llvm.ConstInt(i32, 0, false),
	}, []llvm.BasicBlock{left, right})
	result := builder.CreateAdd(phi, llvm.ConstInt(i32, 1, false), "result")
	builder.CreateRet(result)

	if err := spillValue(ctx, phi, field); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify phi spill: %v\n%s", err, mod.String())
	}
	ir := mod.String()
	if !strings.Contains(ir, "store i32 %selected, ptr %field") ||
		!strings.Contains(ir, "add i32 %selected.reload, 1") {
		t.Fatalf("phi was not canonicalized through the frame:\n%s", ir)
	}
}

func TestSpillValueReloadsPhiOnIncomingEdge(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("spill-phi")
	defer mod.Dispose()

	i1 := ctx.Int1Type()
	i32 := ctx.Int32Type()
	frameType := ctx.StructType([]llvm.Type{i32}, false)
	fn := llvm.AddFunction(mod, "function", llvm.FunctionType(i32, []llvm.Type{
		llvm.PointerType(frameType, 0), i1, i32,
	}, false))
	entry := ctx.AddBasicBlock(fn, "entry")
	left := ctx.AddBasicBlock(fn, "left")
	right := ctx.AddBasicBlock(fn, "right")
	merge := ctx.AddBasicBlock(fn, "merge")
	builder := ctx.NewBuilder()
	defer builder.Dispose()

	builder.SetInsertPointAtEnd(entry)
	field := builder.CreateStructGEP(frameType, fn.Param(0), 0, "field")
	store := builder.CreateStore(fn.Param(2), field)
	builder.CreateCondBr(fn.Param(1), left, right)
	builder.SetInsertPointAtEnd(left)
	builder.CreateBr(merge)
	builder.SetInsertPointAtEnd(right)
	builder.CreateBr(merge)
	builder.SetInsertPointAtEnd(merge)
	phi := builder.CreatePHI(i32, "selected")
	phi.AddIncoming(
		[]llvm.Value{fn.Param(2), llvm.ConstInt(i32, 0, false)},
		[]llvm.BasicBlock{left, right},
	)
	builder.CreateRet(phi)

	replaceValueUsesWithLoads(ctx, fn.Param(2), field, store)
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify phi reload: %v\n%s", err, mod.String())
	}
	if incoming := merge.FirstInstruction().IncomingValue(0); incoming.InstructionParent() != left {
		t.Fatalf("phi reload is not on the incoming edge:\n%s", mod.String())
	}
}

func TestSpillValueReplacesAllocaWithFrameAddress(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("spill-alloca")
	defer mod.Dispose()
	i32 := ctx.Int32Type()
	frameType := ctx.StructType([]llvm.Type{i32}, false)
	fn := llvm.AddFunction(mod, "function", llvm.FunctionType(i32, []llvm.Type{
		llvm.PointerType(frameType, 0),
	}, false))
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	field := builder.CreateStructGEP(frameType, fn.Param(0), 0, "field")
	local := builder.CreateAlloca(i32, "local")
	builder.CreateStore(llvm.ConstInt(i32, 9, false), local)
	builder.CreateRet(builder.CreateLoad(i32, local, "result"))

	if err := spillValue(ctx, local, field); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify alloca frame address: %v\n%s", err, mod.String())
	}
	if strings.Contains(mod.String(), " = alloca ") {
		t.Fatalf("alloca remains after frame replacement:\n%s", mod.String())
	}
}

func TestSpillValueRejectsUnsupportedDefinitions(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("spill-errors")
	defer mod.Dispose()
	i32 := ctx.Int32Type()
	frameType := ctx.StructType([]llvm.Type{i32}, false)
	fn := llvm.AddFunction(mod, "function", llvm.FunctionType(ctx.VoidType(), []llvm.Type{i32}, false))
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	field := builder.CreateStructGEP(frameType, builder.CreateAlloca(frameType, "frame"), 0, "field")
	dynamic := builder.CreateArrayAlloca(i32, fn.Param(0), "dynamic")
	builder.CreateRetVoid()

	if err := spillValue(ctx, dynamic, field); err == nil || !strings.Contains(err.Error(), "separate frame storage") {
		t.Fatalf("dynamic alloca spill error = %v", err)
	}
}

func TestSpillValueStoresOrdinaryCallResult(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("spill-call")
	defer mod.Dispose()
	i32 := ctx.Int32Type()
	frameType := ctx.StructType([]llvm.Type{i32}, false)
	callee := llvm.AddFunction(mod, "callee", llvm.FunctionType(i32, nil, false))
	fn := llvm.AddFunction(mod, "function", llvm.FunctionType(i32, []llvm.Type{
		llvm.PointerType(frameType, 0),
	}, false))
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	field := builder.CreateStructGEP(frameType, fn.Param(0), 0, "field")
	call := builder.CreateCall(callee.GlobalValueType(), callee, nil, "call")
	builder.CreateRet(call)

	if err := spillValue(ctx, call, field); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify spilled call result: %v\n%s", err, mod.String())
	}
	ir := mod.String()
	if !strings.Contains(ir, "store i32 %call, ptr %field") ||
		!strings.Contains(ir, "ret i32 %call.reload") {
		t.Fatalf("ordinary call result was not stored in the frame:\n%s", ir)
	}
}

func TestSpillValueReplacesOverAlignedAlloca(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("spill-aligned-alloca")
	defer mod.Dispose()
	i32 := ctx.Int32Type()
	frameType := ctx.StructType([]llvm.Type{i32}, false)
	fn := llvm.AddFunction(mod, "function", llvm.FunctionType(ctx.VoidType(), nil, false))
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	frame := builder.CreateAlloca(frameType, "frame")
	frame.SetAlignment(32)
	field := builder.CreateStructGEP(frameType, frame, 0, "field")
	local := builder.CreateAlloca(i32, "local")
	local.SetAlignment(32)
	builder.CreateStore(llvm.ConstInt(i32, 9, false), local)
	builder.CreateRetVoid()

	if err := spillValue(ctx, local, field); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify aligned alloca frame address: %v\n%s", err, mod.String())
	}
	if strings.Contains(mod.String(), "%local = alloca") {
		t.Fatalf("over-aligned alloca remains after frame replacement:\n%s", mod.String())
	}
}
