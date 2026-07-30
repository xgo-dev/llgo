package wasmresume

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestLowerUnwindMarkersStoresAndClearsFrameSlot(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("unwind-markers")
	defer mod.Dispose()

	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	entryType := llvm.FunctionType(ctx.Int8Type(), []llvm.Type{ptr, ptr}, false)
	entry := llvm.AddFunction(mod, "resume", entryType)
	block := ctx.AddBasicBlock(entry, "entry")
	handler := ctx.AddBasicBlock(entry, "handler")
	registerType := llvm.FunctionType(ctx.VoidType(), []llvm.Type{ptr, ptr}, false)
	register := llvm.AddFunction(mod, RegisterUnwindSymbol, registerType)
	clearType := llvm.FunctionType(ctx.VoidType(), nil, false)
	clear := llvm.AddFunction(mod, ClearUnwindSymbol, clearType)
	token := llvm.AddGlobal(mod, ctx.Int8Type(), "token")

	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	field := builder.CreateAlloca(ptr, "unwind.slot")
	builder.CreateCall(registerType, register, []llvm.Value{
		token,
		llvm.BlockAddress(entry, handler),
	}, "")
	builder.CreateCall(clearType, clear, nil, "")
	builder.CreateRet(llvm.ConstInt(ctx.Int8Type(), actionReturn, false))
	builder.SetInsertPointAtEnd(handler)
	builder.CreateRet(llvm.ConstInt(ctx.Int8Type(), actionReturn, false))

	plan := framePlan{
		slots:      []frameSlot{{id: 1, kind: slotUnwind, typ: ptr}},
		unwindSlot: 1,
	}
	if err := lowerUnwindMarkers(ctx, entry, plan, field); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyFunction(entry, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify lowered marker function: %v\n%s", err, mod.String())
	}
	ir := mod.String()
	if strings.Contains(ir, "call void @"+RegisterUnwindSymbol) ||
		strings.Contains(ir, "call void @"+ClearUnwindSymbol) {
		t.Fatalf("unwind marker call remains:\n%s", ir)
	}
	for _, want := range []string{
		"store ptr @token, ptr %unwind.slot",
		"store ptr null, ptr %unwind.slot",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("lowered unwind markers are missing %q:\n%s", want, ir)
		}
	}
}

func TestLowerUnwindMarkersValidatesPlan(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	ptr := llvm.PointerType(ctx.Int8Type(), 0)

	t.Run("missing marker", func(t *testing.T) {
		mod := ctx.NewModule("missing-marker")
		defer mod.Dispose()
		entry := llvm.AddFunction(mod, "resume", llvm.FunctionType(ctx.Int8Type(), nil, false))
		block := ctx.AddBasicBlock(entry, "entry")
		builder := ctx.NewBuilder()
		defer builder.Dispose()
		builder.SetInsertPointAtEnd(block)
		field := builder.CreateAlloca(ptr, "unwind.slot")
		builder.CreateRet(llvm.ConstInt(ctx.Int8Type(), actionReturn, false))

		plan := framePlan{
			slots:      []frameSlot{{id: 1, kind: slotUnwind, typ: ptr}},
			unwindSlot: 1,
		}
		if err := lowerUnwindMarkers(ctx, entry, plan, field); err == nil {
			t.Fatal("lowerUnwindMarkers accepted a missing registration")
		}
		if err := lowerUnwindMarkers(ctx, entry, framePlan{}, llvm.Value{}); err != nil {
			t.Fatalf("marker-free frame returned %v", err)
		}
	})

	t.Run("missing slot", func(t *testing.T) {
		mod := ctx.NewModule("missing-slot")
		defer mod.Dispose()
		registerType := llvm.FunctionType(ctx.VoidType(), []llvm.Type{ptr, ptr}, false)
		register := llvm.AddFunction(mod, RegisterUnwindSymbol, registerType)
		entry := llvm.AddFunction(mod, "resume", llvm.FunctionType(ctx.Int8Type(), nil, false))
		block := ctx.AddBasicBlock(entry, "entry")
		builder := ctx.NewBuilder()
		defer builder.Dispose()
		builder.SetInsertPointAtEnd(block)
		builder.CreateCall(registerType, register, []llvm.Value{
			llvm.ConstNull(ptr),
			llvm.ConstNull(ptr),
		}, "")
		builder.CreateRet(llvm.ConstInt(ctx.Int8Type(), actionReturn, false))
		if err := lowerUnwindMarkers(ctx, entry, framePlan{}, llvm.Value{}); err == nil {
			t.Fatal("lowerUnwindMarkers accepted a marker without a slot")
		}
	})

	t.Run("invalid register", func(t *testing.T) {
		mod := ctx.NewModule("invalid-register")
		defer mod.Dispose()
		registerType := llvm.FunctionType(ctx.VoidType(), []llvm.Type{ptr}, false)
		register := llvm.AddFunction(mod, RegisterUnwindSymbol, registerType)
		entry := llvm.AddFunction(mod, "resume", llvm.FunctionType(ctx.Int8Type(), nil, false))
		block := ctx.AddBasicBlock(entry, "entry")
		builder := ctx.NewBuilder()
		defer builder.Dispose()
		builder.SetInsertPointAtEnd(block)
		field := builder.CreateAlloca(ptr, "unwind.slot")
		builder.CreateCall(registerType, register, []llvm.Value{llvm.ConstNull(ptr)}, "")
		builder.CreateRet(llvm.ConstInt(ctx.Int8Type(), actionReturn, false))
		plan := framePlan{
			slots:      []frameSlot{{id: 1, kind: slotUnwind, typ: ptr}},
			unwindSlot: 1,
		}
		if err := lowerUnwindMarkers(ctx, entry, plan, field); err == nil {
			t.Fatal("lowerUnwindMarkers accepted an invalid registration")
		}
	})
}

func TestFindUnwindPlan(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	registerType := llvm.FunctionType(ctx.VoidType(), []llvm.Type{ptr, ptr}, false)

	newFunction := func(mod llvm.Module, handler llvm.Value) (llvm.Value, llvm.Value) {
		register := llvm.AddFunction(mod, RegisterUnwindSymbol, registerType)
		fn := llvm.AddFunction(mod, "f", llvm.FunctionType(ctx.VoidType(), nil, false))
		entry := ctx.AddBasicBlock(fn, "entry")
		target := ctx.AddBasicBlock(fn, "target")
		builder := ctx.NewBuilder()
		defer builder.Dispose()
		builder.SetInsertPointAtEnd(entry)
		call := builder.CreateCall(registerType, register, []llvm.Value{
			llvm.ConstNull(ptr),
			handler,
		}, "")
		builder.CreateBr(target)
		builder.SetInsertPointAtEnd(target)
		builder.CreateRetVoid()
		return fn, call
	}

	mod := ctx.NewModule("valid-unwind")
	fn := llvm.AddFunction(mod, "f", llvm.FunctionType(ctx.VoidType(), nil, false))
	entry := ctx.AddBasicBlock(fn, "entry")
	target := ctx.AddBasicBlock(fn, "target")
	register := llvm.AddFunction(mod, RegisterUnwindSymbol, registerType)
	builder := ctx.NewBuilder()
	builder.SetInsertPointAtEnd(entry)
	builder.CreateCall(registerType, register, []llvm.Value{
		llvm.ConstNull(ptr),
		llvm.BlockAddress(fn, target),
	}, "")
	builder.CreateBr(target)
	builder.SetInsertPointAtEnd(target)
	builder.CreateRetVoid()
	builder.Dispose()
	plan, err := findUnwindPlan(fn)
	if err != nil || plan.block != target || plan.typ != ptr {
		t.Fatalf("findUnwindPlan = %+v, %v", plan, err)
	}
	mod.Dispose()

	mod = ctx.NewModule("invalid-unwind")
	fn, call := newFunction(mod, llvm.ConstNull(ptr))
	if _, err := findUnwindPlan(fn); err == nil {
		t.Fatal("findUnwindPlan accepted a non-block handler")
	}
	call.EraseFromParentAsInstruction()
	mod.Dispose()
}

func TestUnwindOnlyFunctionUsesStateMachine(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("unwind-only")
	defer mod.Dispose()
	mod.SetTarget("wasm32-unknown-unknown")
	mod.SetDataLayout("e-m:e-p:32:32-i64:64-n32:64-S128")
	targetData := llvm.NewTargetData(mod.DataLayout())
	defer targetData.Dispose()

	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	registerType := llvm.FunctionType(ctx.VoidType(), []llvm.Type{ptr, ptr}, false)
	register := llvm.AddFunction(mod, RegisterUnwindSymbol, registerType)
	clearType := llvm.FunctionType(ctx.VoidType(), nil, false)
	clear := llvm.AddFunction(mod, ClearUnwindSymbol, clearType)
	fn := llvm.AddFunction(mod, "with.c.defer", llvm.FunctionType(ctx.VoidType(), nil, false))
	markFunction(ctx, fn)
	entry := ctx.AddBasicBlock(fn, "entry")
	handler := ctx.AddBasicBlock(fn, "handler")
	done := ctx.AddBasicBlock(fn, "done")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(entry)
	builder.CreateCall(registerType, register, []llvm.Value{
		llvm.ConstNull(ptr),
		llvm.BlockAddress(fn, handler),
	}, "")
	builder.CreateBr(done)
	builder.SetInsertPointAtEnd(handler)
	builder.CreateCall(clearType, clear, nil, "")
	builder.CreateRetVoid()
	builder.SetInsertPointAtEnd(done)
	builder.CreateCall(clearType, clear, nil, "")
	builder.CreateRetVoid()

	if err := Lower(mod, targetData); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify unwind-only state machine: %v\n%s", err, mod.String())
	}
	ir := mod.String()
	for _, want := range []string{
		"define internal i8 @__llgo_wasm_resume.with.c.defer",
		"define void @with.c.defer()",
		"i32 1, label %handler",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("unwind-only state machine is missing %q:\n%s", want, ir)
		}
	}
	if strings.Contains(ir, "call void @"+RegisterUnwindSymbol) ||
		strings.Contains(ir, "call void @"+ClearUnwindSymbol) {
		t.Fatalf("unwind marker remains in unwind-only state machine:\n%s", ir)
	}
}
