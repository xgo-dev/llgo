package wasmresume

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestLowerPrototypeExecutesDirectCallStateMachine(t *testing.T) {
	llvm.LinkInMCJIT()
	if err := llvm.InitializeNativeTarget(); err != nil {
		t.Fatal(err)
	}
	if err := llvm.InitializeNativeAsmPrinter(); err != nil {
		t.Fatal(err)
	}

	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("state-execution")
	moduleOwned := true
	defer func() {
		if moduleOwned {
			mod.Dispose()
		}
	}()

	triple := llvm.DefaultTargetTriple()
	target, err := llvm.GetTargetFromTriple(triple)
	if err != nil {
		t.Fatal(err)
	}
	machine := target.CreateTargetMachine(
		triple, "", "", llvm.CodeGenLevelNone, llvm.RelocDefault, llvm.CodeModelJITDefault,
	)
	defer machine.Dispose()
	targetData := machine.CreateTargetData()
	defer targetData.Dispose()
	mod.SetTarget(triple)
	mod.SetDataLayout(targetData.String())

	i32 := ctx.Int32Type()
	sig := llvm.FunctionType(i32, []llvm.Type{i32}, false)
	callee := llvm.AddFunction(mod, "callee", sig)
	markFunction(ctx, callee)
	calleeBlock := ctx.AddBasicBlock(callee, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(calleeBlock)
	builder.CreateRet(builder.CreateAdd(callee.Param(0), llvm.ConstInt(i32, 1, false), "sum"))

	middle := llvm.AddFunction(mod, "middle", sig)
	markFunction(ctx, middle)
	middleBlock := ctx.AddBasicBlock(middle, "entry")
	builder.SetInsertPointAtEnd(middleBlock)
	first := builder.CreateCall(sig, callee, []llvm.Value{middle.Param(0)}, "first")
	markCall(ctx, first)
	second := builder.CreateCall(sig, callee, []llvm.Value{first}, "second")
	markCall(ctx, second)
	builder.CreateRet(builder.CreateMul(second, llvm.ConstInt(i32, 2, false), "middle.result"))

	caller := llvm.AddFunction(mod, "caller", sig)
	markFunction(ctx, caller)
	callerBlock := ctx.AddBasicBlock(caller, "entry")
	builder.SetInsertPointAtEnd(callerBlock)
	before := builder.CreateAdd(caller.Param(0), llvm.ConstInt(i32, 2, false), "before")
	call := builder.CreateCall(sig, middle, []llvm.Value{before}, "called")
	markCall(ctx, call)
	builder.CreateRet(builder.CreateMul(before, call, "result"))

	lowered, err := lowerPrototype(mod, targetData)
	if err != nil {
		t.Fatal(err)
	}
	if len(lowered) != 2 {
		t.Fatalf("lowered states = %d, want 2", len(lowered))
	}
	var root loweredState
	for _, state := range lowered {
		if state.layout.plan.function.Name() == "caller" {
			root = state
			break
		}
	}
	if root.entry.IsNil() {
		t.Fatal("caller state machine was not lowered")
	}
	harness := defineStateMachineHarness(
		mod, targetData, root, []llvm.Value{llvm.ConstInt(i32, 5, false)},
	)
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify executable state machine: %v\n%s", err, mod.String())
	}

	options := llvm.NewMCJITCompilerOptions()
	options.SetMCJITOptimizationLevel(0)
	engine, err := llvm.NewMCJITCompiler(mod, options)
	if err != nil {
		t.Fatal(err)
	}
	moduleOwned = false
	defer engine.Dispose()

	result := engine.RunFunction(harness, nil)
	defer result.Dispose()
	if got := result.Int(true); got != 126 {
		t.Fatalf("state machine result = %d, want 126", got)
	}

	arg := llvm.NewGenericValueFromInt(i32, 5, true)
	defer arg.Dispose()
	result = engine.RunFunction(caller, []llvm.GenericValue{arg})
	defer result.Dispose()
	if got := result.Int(true); got != 126 {
		t.Fatalf("compatibility wrapper result = %d, want 126", got)
	}
}

func TestLowerPrototypeBuildsDirectCallStateMachine(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("state")
	defer mod.Dispose()
	mod.SetDataLayout("e-m:e-p:32:32-i64:64-n32:64-S128")
	targetData := llvm.NewTargetData(mod.DataLayout())
	defer targetData.Dispose()

	i32 := ctx.Int32Type()
	sig := llvm.FunctionType(i32, []llvm.Type{i32}, false)
	callee := llvm.AddFunction(mod, "callee", sig)
	markFunction(ctx, callee)
	calleeBlock := ctx.AddBasicBlock(callee, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(calleeBlock)
	builder.CreateRet(builder.CreateAdd(callee.Param(0), llvm.ConstInt(i32, 1, false), "sum"))

	caller := llvm.AddFunction(mod, "caller", sig)
	markFunction(ctx, caller)
	caller.Param(0).SetName("input")
	callerBlock := ctx.AddBasicBlock(caller, "entry")
	builder.SetInsertPointAtEnd(callerBlock)
	before := builder.CreateAdd(caller.Param(0), llvm.ConstInt(i32, 2, false), "before")
	call := builder.CreateCall(sig, callee, []llvm.Value{before}, "called")
	markCall(ctx, call)
	result := builder.CreateMul(before, call, "result")
	builder.CreateRet(result)

	lowered, err := lowerPrototype(mod, targetData)
	if err != nil {
		t.Fatal(err)
	}
	if len(lowered) != 1 || lowered[0].layout.plan.function != caller {
		t.Fatalf("lowered states = %+v", lowered)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify state machine: %v\n%s", err, mod.String())
	}

	ir := mod.String()
	for _, want := range []string{
		`@__llgo_wasm_resume_desc.callee = constant`,
		`@__llgo_wasm_resume_desc.caller = constant`,
		`define internal i8 @__llgo_wasm_resume.caller`,
		`switch i32 %pc, label %invalid-pc [`,
		`i32 0, label %entry`,
		`i32 1, label %resume.1`,
		`call ptr @__llgo_wasm_resume_alloc(ptr %0,`,
		`call void @llvm.memset`,
		`ret i8 0`,
		`%returned = load ptr`,
		`call void @__llgo_wasm_resume_free(ptr %0, ptr %returned)`,
		`ret i8 1`,
	} {
		if !strings.Contains(ir, want) {
			t.Errorf("state machine is missing %q:\n%s", want, ir)
		}
	}
}

func TestLowerEmitsWasmObject(t *testing.T) {
	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllAsmPrinters()

	for _, triple := range []string{"wasm32-unknown-unknown", "wasm64-unknown-unknown"} {
		t.Run(triple, func(t *testing.T) {
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			mod := ctx.NewModule(triple)
			defer mod.Dispose()

			target, err := llvm.GetTargetFromTriple(triple)
			if err != nil {
				t.Fatal(err)
			}
			machine := target.CreateTargetMachine(
				triple, "", "", llvm.CodeGenLevelNone, llvm.RelocDefault, llvm.CodeModelDefault,
			)
			defer machine.Dispose()
			targetData := machine.CreateTargetData()
			defer targetData.Dispose()
			mod.SetTarget(triple)
			mod.SetDataLayout(targetData.String())

			i32 := ctx.Int32Type()
			sig := llvm.FunctionType(i32, []llvm.Type{i32}, false)
			callee := llvm.AddFunction(mod, "callee", sig)
			markFunction(ctx, callee)
			calleeBlock := ctx.AddBasicBlock(callee, "entry")
			builder := ctx.NewBuilder()
			defer builder.Dispose()
			builder.SetInsertPointAtEnd(calleeBlock)
			builder.CreateRet(callee.Param(0))

			ptr := llvm.PointerType(ctx.Int8Type(), 0)
			callerType := llvm.FunctionType(i32, []llvm.Type{ptr, i32}, false)
			caller := llvm.AddFunction(mod, "caller", callerType)
			markFunction(ctx, caller)
			callerBlock := ctx.AddBasicBlock(caller, "entry")
			builder.SetInsertPointAtEnd(callerBlock)
			call := builder.CreateCall(sig, caller.Param(0), []llvm.Value{caller.Param(1)}, "called")
			markCall(ctx, call)
			builder.CreateRet(call)

			if err := Lower(mod, targetData); err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("verify %s state machine: %v\n%s", triple, err, mod.String())
			}
			object, err := machine.EmitToMemoryBuffer(mod, llvm.ObjectFile)
			if err != nil {
				t.Fatalf("emit %s state machine: %v\n%s", triple, err, mod.String())
			}
			defer object.Dispose()
			if data := object.Bytes(); len(data) < 4 || string(data[:4]) != "\x00asm" {
				t.Fatalf("%s object does not have the WebAssembly header", triple)
			}
		})
	}
}

func TestLowerRejectsNonWasmTarget(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("native")
	defer mod.Dispose()
	mod.SetTarget("aarch64-unknown-linux-gnu")
	targetData := llvm.NewTargetData("e-m:e-p:64:64-i64:64-n32:64-S128")
	defer targetData.Dispose()

	if err := Lower(mod, targetData); err == nil ||
		!strings.Contains(err.Error(), "is not WebAssembly") {
		t.Fatalf("Lower error = %v", err)
	}
}

func TestLowerPrototypeExecutesIndirectStart(t *testing.T) {
	llvm.LinkInMCJIT()
	if err := llvm.InitializeNativeTarget(); err != nil {
		t.Fatal(err)
	}
	if err := llvm.InitializeNativeAsmPrinter(); err != nil {
		t.Fatal(err)
	}

	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("indirect-execution")
	moduleOwned := true
	defer func() {
		if moduleOwned {
			mod.Dispose()
		}
	}()

	triple := llvm.DefaultTargetTriple()
	target, err := llvm.GetTargetFromTriple(triple)
	if err != nil {
		t.Fatal(err)
	}
	machine := target.CreateTargetMachine(
		triple, "", "", llvm.CodeGenLevelNone, llvm.RelocDefault, llvm.CodeModelJITDefault,
	)
	defer machine.Dispose()
	targetData := machine.CreateTargetData()
	defer targetData.Dispose()
	mod.SetTarget(triple)
	mod.SetDataLayout(targetData.String())

	i32 := ctx.Int32Type()
	sig := llvm.FunctionType(i32, []llvm.Type{i32}, false)
	callee := llvm.AddFunction(mod, "callee", sig)
	markFunction(ctx, callee)
	block := ctx.AddBasicBlock(callee, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	builder.CreateRet(builder.CreateAdd(callee.Param(0), llvm.ConstInt(i32, 1, false), "result"))

	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	startType := llvm.FunctionType(ptr, []llvm.Type{ptr, i32}, false)
	start := llvm.AddFunction(mod, StartSymbol(callee.Name()), startType)
	suspend := llvm.AddFunction(mod, SuspendSymbol, llvm.FunctionType(ctx.VoidType(), nil, false))
	markFunction(ctx, suspend)
	callerType := llvm.FunctionType(i32, []llvm.Type{ptr, i32}, false)
	caller := llvm.AddFunction(mod, "caller", callerType)
	markFunction(ctx, caller)
	block = ctx.AddBasicBlock(caller, "entry")
	builder.SetInsertPointAtEnd(block)
	dynamicCall := builder.CreateCall(sig, caller.Param(0), []llvm.Value{caller.Param(1)}, "dynamic")
	markCall(ctx, dynamicCall)
	constantCall := builder.CreateCall(sig, start, []llvm.Value{dynamicCall}, "constant")
	markCall(ctx, constantCall)
	suspendCall := builder.CreateCall(suspend.GlobalValueType(), suspend, nil, "")
	markCall(ctx, suspendCall)
	builder.CreateRet(builder.CreateMul(constantCall, llvm.ConstInt(i32, 2, false), "result"))

	lowered, err := lowerPrototype(mod, targetData)
	if err != nil {
		t.Fatal(err)
	}
	var root loweredState
	for _, state := range lowered {
		if state.layout.plan.function == caller {
			root = state
			break
		}
	}
	if root.entry.IsNil() || start.IsNil() {
		t.Fatal("indirect state machine entries were not emitted")
	}
	harness := defineStateMachineHarness(mod, targetData, root, []llvm.Value{
		start, llvm.ConstInt(i32, 5, false),
	})
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify indirect state machine: %v\n%s", err, mod.String())
	}

	options := llvm.NewMCJITCompilerOptions()
	options.SetMCJITOptimizationLevel(0)
	engine, err := llvm.NewMCJITCompiler(mod, options)
	if err != nil {
		t.Fatal(err)
	}
	moduleOwned = false
	defer engine.Dispose()

	result := engine.RunFunction(harness, nil)
	defer result.Dispose()
	if got := result.Int(true); got != 14 {
		t.Fatalf("indirect state machine result = %d, want 14", got)
	}
}

func defineStateMachineHarness(
	mod llvm.Module, targetData llvm.TargetData, lowered loweredState, params []llvm.Value,
) llvm.Value {
	ctx := mod.Context()
	abi := newResumeABI(ctx, targetData)
	i8 := ctx.Int8Type()
	i32 := ctx.Int32Type()

	childStorageType := llvm.ArrayType(i8, 4096)
	childStorage := llvm.AddGlobal(mod, childStorageType, "child.storage")
	childStorage.SetInitializer(llvm.ConstNull(childStorageType))
	childStorage.SetAlignment(16)
	childOffset := llvm.AddGlobal(mod, i32, "child.offset")
	childOffset.SetInitializer(llvm.ConstInt(i32, 0, false))

	alloc := mod.NamedFunction(frameAllocName)
	block := ctx.AddBasicBlock(alloc, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	offset := builder.CreateLoad(i32, childOffset, "offset")
	builder.CreateStore(
		builder.CreateAdd(offset, llvm.ConstInt(i32, 256, false), ""),
		childOffset,
	)
	builder.CreateRet(builder.CreateInBoundsGEP(i8, childStorage, []llvm.Value{offset}, "frame"))

	free := mod.NamedFunction(frameFreeName)
	block = ctx.AddBasicBlock(free, "entry")
	builder.SetInsertPointAtEnd(block)
	builder.CreateRetVoid()

	close := mod.NamedFunction(frameCloseName)
	block = ctx.AddBasicBlock(close, "entry")
	builder.SetInsertPointAtEnd(block)
	builder.CreateRetVoid()

	root := llvm.AddGlobal(mod, lowered.layout.typ, "root.frame")
	root.SetInitializer(llvm.ConstNull(lowered.layout.typ))
	root.SetAlignment(lowered.layout.alignment)
	context := llvm.AddGlobal(mod, abi.contextType, "resume.context")
	context.SetInitializer(llvm.ConstNull(abi.contextType))

	run := llvm.AddFunction(mod, "run.state.machine", llvm.FunctionType(i32, nil, false))
	entryBlock := ctx.AddBasicBlock(run, "entry")
	loopBlock := ctx.AddBasicBlock(run, "loop")
	resumeBlock := ctx.AddBasicBlock(run, "resume")
	popBlock := ctx.AddBasicBlock(run, "pop")
	doneBlock := ctx.AddBasicBlock(run, "done")
	failedBlock := ctx.AddBasicBlock(run, "failed")

	builder.SetInsertPointAtEnd(entryBlock)
	builder.CreateStore(
		llvm.ConstNull(abi.ptr),
		builder.CreateStructGEP(lowered.layout.typ, root, 0, ""),
	)
	builder.CreateStore(
		lowered.descriptor,
		builder.CreateStructGEP(lowered.layout.typ, root, 1, ""),
	)
	builder.CreateStore(
		llvm.ConstInt(i32, 0, false),
		builder.CreateStructGEP(lowered.layout.typ, root, 2, ""),
	)
	param := 0
	for _, slot := range lowered.layout.plan.slots {
		if slot.kind == slotParameter {
			builder.CreateStore(
				params[param],
				builder.CreateStructGEP(
					lowered.layout.typ, root, lowered.layout.fieldIndex(slot.id), "",
				),
			)
			param++
		}
	}
	builder.CreateStore(root, builder.CreateStructGEP(abi.contextType, context, 0, ""))
	builder.CreateStore(
		llvm.ConstNull(abi.ptr),
		builder.CreateStructGEP(abi.contextType, context, 1, ""),
	)
	builder.CreateBr(loopBlock)

	builder.SetInsertPointAtEnd(loopBlock)
	topField := builder.CreateStructGEP(abi.contextType, context, 0, "")
	top := builder.CreateLoad(abi.ptr, topField, "top")
	builder.CreateCondBr(
		builder.CreateICmp(llvm.IntNE, top, llvm.ConstNull(abi.ptr), ""),
		resumeBlock,
		doneBlock,
	)

	framePrefix := ctx.StructType([]llvm.Type{abi.ptr, abi.ptr, i32}, false)
	builder.SetInsertPointAtEnd(resumeBlock)
	descriptor := builder.CreateLoad(
		abi.ptr, builder.CreateStructGEP(framePrefix, top, 1, ""), "descriptor",
	)
	resume := builder.CreateLoad(
		abi.ptr, builder.CreateStructGEP(abi.descriptorType, descriptor, 0, ""), "resume.entry",
	)
	action := builder.CreateCall(abi.entryType, resume, []llvm.Value{context, top}, "action")
	switchAction := builder.CreateSwitch(action, failedBlock, 3)
	switchAction.AddCase(llvm.ConstInt(i8, actionContinue, false), loopBlock)
	switchAction.AddCase(llvm.ConstInt(i8, actionReturn, false), popBlock)
	switchAction.AddCase(llvm.ConstInt(i8, actionSuspend, false), loopBlock)

	builder.SetInsertPointAtEnd(popBlock)
	parent := builder.CreateLoad(
		abi.ptr, builder.CreateStructGEP(framePrefix, top, 0, ""), "parent",
	)
	builder.CreateStore(parent, topField)
	builder.CreateStore(top, builder.CreateStructGEP(abi.contextType, context, 1, ""))
	builder.CreateBr(loopBlock)

	builder.SetInsertPointAtEnd(doneBlock)
	builder.CreateRet(builder.CreateLoad(
		i32,
		builder.CreateStructGEP(
			lowered.layout.typ, root,
			lowered.layout.fieldIndex(lowered.layout.plan.resultSlot), "",
		),
		"result",
	))

	builder.SetInsertPointAtEnd(failedBlock)
	builder.CreateRet(llvm.ConstInt(i32, ^uint64(0), true))
	return run
}

func TestLowerPrototypeSupportsDynamicAlloca(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("dynamic")
	defer mod.Dispose()
	targetData := llvm.NewTargetData("e-m:e-p:32:32-i64:64-n32:64-S128")
	defer targetData.Dispose()

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

	if _, err := lowerPrototype(mod, targetData); err != nil {
		t.Fatal(err)
	}
	if ir := mod.String(); strings.Contains(ir, "%local = alloca") ||
		!strings.Contains(ir, "@__llgo_wasm_resume_alloc_dynamic") {
		t.Fatalf("dynamic alloca was not lowered:\n%s", ir)
	}
}

func TestLowerPrototypeRejectsVariadicResumeCall(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("variadic")
	defer mod.Dispose()
	targetData := llvm.NewTargetData("e-m:e-p:32:32-i64:64-n32:64-S128")
	defer targetData.Dispose()

	i32 := ctx.Int32Type()
	variadicType := llvm.FunctionType(ctx.VoidType(), []llvm.Type{i32}, true)
	callee := llvm.AddFunction(mod, "callee", variadicType)
	fn := llvm.AddFunction(mod, "caller", llvm.FunctionType(ctx.VoidType(), nil, false))
	markFunction(ctx, fn)
	block := ctx.AddBasicBlock(fn, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	call := builder.CreateCall(
		variadicType, callee, []llvm.Value{llvm.ConstInt(i32, 1, false)}, "",
	)
	markCall(ctx, call)
	builder.CreateRetVoid()

	if _, err := lowerPrototype(mod, targetData); err == nil ||
		!strings.Contains(err.Error(), "variadic") {
		t.Fatalf("lowerPrototype error = %v", err)
	}
}
