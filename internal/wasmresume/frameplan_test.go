package wasmresume

import (
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestPlanFramesStraightLineValues(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("straight")
	defer mod.Dispose()

	i32 := ctx.Int32Type()
	callee := llvm.AddFunction(mod, "callee", llvm.FunctionType(i32, []llvm.Type{i32}, false))
	fn := llvm.AddFunction(mod, "caller", llvm.FunctionType(i32, []llvm.Type{i32}, false))
	markFunction(ctx, fn)
	fn.Param(0).SetName("input")
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	before := builder.CreateAdd(fn.Param(0), llvm.ConstInt(i32, 1, false), "before")
	call := builder.CreateCall(callee.GlobalValueType(), callee, []llvm.Value{before}, "result")
	markCall(ctx, call)
	after := builder.CreateAdd(before, call, "after")
	builder.CreateRet(after)

	plans, err := planFrames(mod)
	if err != nil {
		t.Fatal(err)
	}
	plan := onlyFramePlan(t, plans)
	assertSlots(t, plan, []slotWant{
		{name: "input", kind: slotParameter},
		{kind: slotFunctionResult},
		{name: "before", kind: slotValue},
		{name: "result", kind: slotValue},
	})
	if len(plan.calls) != 1 {
		t.Fatalf("calls = %+v", plan.calls)
	}
	if got, want := plan.calls[0].live, []uint32{3}; !equalIDs(got, want) {
		t.Fatalf("live slots = %v, want %v", got, want)
	}
	if plan.resultSlot != 2 {
		t.Fatalf("function result slot = %d, want 2", plan.resultSlot)
	}
	if plan.calls[0].resultSlot != 4 {
		t.Fatalf("call result slot = %d, want 4", plan.calls[0].resultSlot)
	}
}

func TestPlanFramesKeepsParameterAndAllocaAcrossCall(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("alloca")
	defer mod.Dispose()

	i32 := ctx.Int32Type()
	voidFn := llvm.FunctionType(ctx.VoidType(), nil, false)
	callee := llvm.AddFunction(mod, "callee", voidFn)
	fn := llvm.AddFunction(mod, "caller", llvm.FunctionType(i32, []llvm.Type{i32}, false))
	markFunction(ctx, fn)
	fn.Param(0).SetName("input")
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	local := builder.CreateAlloca(i32, "local")
	builder.CreateStore(fn.Param(0), local)
	call := builder.CreateCall(voidFn, callee, nil, "")
	markCall(ctx, call)
	loaded := builder.CreateLoad(i32, local, "loaded")
	after := builder.CreateAdd(fn.Param(0), loaded, "after")
	builder.CreateRet(after)

	plans, err := planFrames(mod)
	if err != nil {
		t.Fatal(err)
	}
	plan := onlyFramePlan(t, plans)
	assertSlots(t, plan, []slotWant{
		{name: "input", kind: slotParameter},
		{kind: slotFunctionResult},
		{name: "local", kind: slotAlloca},
	})
	if got, want := plan.calls[0].live, []uint32{1, 3}; !equalIDs(got, want) {
		t.Fatalf("live slots = %v, want %v", got, want)
	}
	if plan.calls[0].resultSlot != 0 {
		t.Fatalf("void call result slot = %d, want 0", plan.calls[0].resultSlot)
	}
	if plan.slots[2].typ != i32 || plan.slots[2].dynamic {
		t.Fatalf("static alloca slot = %+v, want embedded i32", plan.slots[2])
	}
}

func TestPlanFramesKeepsAllocaReferencedOnlyByCallArgument(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("alloca-argument")
	defer mod.Dispose()

	i32 := ctx.Int32Type()
	ptr := llvm.PointerType(i32, 0)
	calleeType := llvm.FunctionType(ctx.VoidType(), []llvm.Type{ptr}, false)
	callee := llvm.AddFunction(mod, "callee", calleeType)
	fn := llvm.AddFunction(mod, "caller", llvm.FunctionType(ctx.VoidType(), nil, false))
	markFunction(ctx, fn)
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	local := builder.CreateAlloca(i32, "local")
	derived := builder.CreateGEP(i32, local, []llvm.Value{llvm.ConstInt(ctx.Int32Type(), 0, false)}, "derived")
	call := builder.CreateCall(calleeType, callee, []llvm.Value{derived}, "")
	markCall(ctx, call)
	builder.CreateRetVoid()

	plans, err := planFrames(mod)
	if err != nil {
		t.Fatal(err)
	}
	plan := onlyFramePlan(t, plans)
	assertSlots(t, plan, []slotWant{{name: "local", kind: slotAlloca}})
	if got, want := plan.calls[0].live, []uint32{1}; !equalIDs(got, want) {
		t.Fatalf("live slots = %v, want %v", got, want)
	}
}

func TestPlanFramesMarksDynamicAllocaStorage(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("dynamic-alloca")
	defer mod.Dispose()

	i32 := ctx.Int32Type()
	ptr := llvm.PointerType(i32, 0)
	calleeType := llvm.FunctionType(ctx.VoidType(), []llvm.Type{ptr}, false)
	callee := llvm.AddFunction(mod, "callee", calleeType)
	fn := llvm.AddFunction(mod, "caller", llvm.FunctionType(ctx.VoidType(), []llvm.Type{i32}, false))
	markFunction(ctx, fn)
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	local := builder.CreateArrayAlloca(i32, fn.Param(0), "local")
	call := builder.CreateCall(calleeType, callee, []llvm.Value{local}, "")
	markCall(ctx, call)
	builder.CreateRetVoid()

	plans, err := planFrames(mod)
	if err != nil {
		t.Fatal(err)
	}
	plan := onlyFramePlan(t, plans)
	assertSlots(t, plan, []slotWant{
		{kind: slotParameter},
		{name: "local", kind: slotAlloca},
	})
	if plan.slots[1].typ != ptr || !plan.slots[1].dynamic {
		t.Fatalf("dynamic alloca slot = %+v, want pointer storage", plan.slots[1])
	}
}

func TestPlanFramesDoesNotPersistCopiedCallArguments(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("copied-arguments")
	defer mod.Dispose()

	i32 := ctx.Int32Type()
	calleeType := llvm.FunctionType(ctx.VoidType(), []llvm.Type{i32, i32}, false)
	callee := llvm.AddFunction(mod, "callee", calleeType)
	fn := llvm.AddFunction(mod, "caller", llvm.FunctionType(ctx.VoidType(), []llvm.Type{i32}, false))
	markFunction(ctx, fn)
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	local := builder.CreateAlloca(i32, "local")
	builder.CreateStore(llvm.ConstInt(i32, 7, false), local)
	loaded := builder.CreateLoad(i32, local, "loaded")
	call := builder.CreateCall(calleeType, callee, []llvm.Value{fn.Param(0), loaded}, "")
	markCall(ctx, call)
	builder.CreateRetVoid()

	plans, err := planFrames(mod)
	if err != nil {
		t.Fatal(err)
	}
	plan := onlyFramePlan(t, plans)
	assertSlots(t, plan, []slotWant{{kind: slotParameter}})
	if len(plan.calls[0].live) != 0 {
		t.Fatalf("copied arguments created persistent slots: %+v", plan)
	}
}

func TestPlanFramesTracksPhiUseOnPredecessorEdge(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("phi")
	defer mod.Dispose()

	i1 := ctx.Int1Type()
	i32 := ctx.Int32Type()
	voidFn := llvm.FunctionType(ctx.VoidType(), nil, false)
	callee := llvm.AddFunction(mod, "callee", voidFn)
	fn := llvm.AddFunction(mod, "caller", llvm.FunctionType(i32, []llvm.Type{i1, i32}, false))
	markFunction(ctx, fn)
	fn.Param(1).SetName("input")
	entry := ctx.AddBasicBlock(fn, "entry")
	left := ctx.AddBasicBlock(fn, "left")
	right := ctx.AddBasicBlock(fn, "right")
	merge := ctx.AddBasicBlock(fn, "merge")
	builder := ctx.NewBuilder()
	defer builder.Dispose()

	builder.SetInsertPointAtEnd(entry)
	builder.CreateCondBr(fn.Param(0), left, right)
	builder.SetInsertPointAtEnd(left)
	call := builder.CreateCall(voidFn, callee, nil, "")
	markCall(ctx, call)
	builder.CreateBr(merge)
	builder.SetInsertPointAtEnd(right)
	builder.CreateBr(merge)
	builder.SetInsertPointAtEnd(merge)
	phi := builder.CreatePHI(i32, "selected")
	phi.AddIncoming(
		[]llvm.Value{fn.Param(1), llvm.ConstInt(i32, 0, false)},
		[]llvm.BasicBlock{left, right},
	)
	builder.CreateRet(phi)

	plans, err := planFrames(mod)
	if err != nil {
		t.Fatal(err)
	}
	plan := onlyFramePlan(t, plans)
	assertSlots(t, plan, []slotWant{
		{kind: slotParameter},
		{name: "input", kind: slotParameter},
		{kind: slotFunctionResult},
	})
	if got, want := plan.calls[0].live, []uint32{2}; !equalIDs(got, want) {
		t.Fatalf("live slots = %v, want %v", got, want)
	}
}

func TestPlanFramesIncludesMarkedLeaf(t *testing.T) {
	ctx, mod, _, builder := newInventoryTestFunction(t, true)
	defer ctx.Dispose()
	defer mod.Dispose()
	defer builder.Dispose()
	builder.CreateRetVoid()

	plans, err := planFrames(mod)
	if err != nil {
		t.Fatal(err)
	}
	plan := onlyFramePlan(t, plans)
	if len(plan.slots) != 0 || len(plan.calls) != 0 {
		t.Fatalf("leaf plan = %+v", plan)
	}
}

func TestPlanFramesReservesLeafParametersAndResult(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("leaf-abi")
	defer mod.Dispose()

	i32 := ctx.Int32Type()
	fn := llvm.AddFunction(mod, "leaf", llvm.FunctionType(i32, []llvm.Type{i32}, false))
	markFunction(ctx, fn)
	fn.Param(0).SetName("input")
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	builder.CreateRet(fn.Param(0))

	plans, err := planFrames(mod)
	if err != nil {
		t.Fatal(err)
	}
	plan := onlyFramePlan(t, plans)
	assertSlots(t, plan, []slotWant{
		{name: "input", kind: slotParameter},
		{kind: slotFunctionResult},
	})
	if plan.resultSlot != 2 {
		t.Fatalf("result slot = %d, want 2", plan.resultSlot)
	}
}

func TestPlanFramesOrdersCallsByResumeID(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("order")
	defer mod.Dispose()

	voidFn := llvm.FunctionType(ctx.VoidType(), nil, false)
	callee := llvm.AddFunction(mod, "callee", voidFn)
	fn := llvm.AddFunction(mod, "caller", voidFn)
	markFunction(ctx, fn)
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	first := builder.CreateCall(voidFn, callee, nil, "")
	markCall(ctx, first)
	second := builder.CreateCall(voidFn, callee, nil, "")
	markCall(ctx, second)
	builder.CreateRetVoid()

	plans, err := planFrames(mod)
	if err != nil {
		t.Fatal(err)
	}
	calls := onlyFramePlan(t, plans).calls
	if len(calls) != 2 || calls[0].id != 1 || calls[1].id != 2 {
		t.Fatalf("calls = %+v", calls)
	}
}

type slotWant struct {
	name string
	kind slotKind
}

func onlyFramePlan(t *testing.T, plans []framePlan) framePlan {
	t.Helper()
	if len(plans) != 1 {
		t.Fatalf("plans = %+v", plans)
	}
	return plans[0]
}

func assertSlots(t *testing.T, plan framePlan, want []slotWant) {
	t.Helper()
	if len(plan.slots) != len(want) {
		t.Fatalf("slots = %+v, want %+v", plan.slots, want)
	}
	for i, slot := range plan.slots {
		name := ""
		if !slot.value.IsNil() {
			name = slot.value.Name()
		}
		if slot.id != uint32(i+1) || name != want[i].name || slot.kind != want[i].kind {
			t.Fatalf("slot %d = {id:%d name:%q kind:%d}, want {id:%d name:%q kind:%d}",
				i, slot.id, name, slot.kind, i+1, want[i].name, want[i].kind)
		}
	}
}

func equalIDs(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func markFunction(ctx llvm.Context, fn llvm.Value) {
	fn.AddFunctionAttr(ctx.CreateStringAttribute(FunctionAttribute, "1"))
}
