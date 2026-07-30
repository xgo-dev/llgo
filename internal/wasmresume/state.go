/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package wasmresume

import (
	"fmt"
	"strings"

	"github.com/xgo-dev/llvm"
)

const (
	frameAllocName        = "__llgo_wasm_resume_alloc"
	frameDynamicAllocName = "__llgo_wasm_resume_alloc_dynamic"
	frameFreeName         = "__llgo_wasm_resume_free"
)

type loweredState struct {
	layout     frameLayout
	entry      llvm.Value
	descriptor llvm.Value
}

// Lower replaces marked Go functions and calls with the experimental
// WebAssembly resumable ABI.
func Lower(mod llvm.Module, targetData llvm.TargetData) error {
	triple := mod.Target()
	if !strings.HasPrefix(triple, "wasm32-") && !strings.HasPrefix(triple, "wasm64-") {
		return fmt.Errorf("wasmresume: target %q is not WebAssembly", triple)
	}
	_, err := lowerPrototype(mod, targetData)
	return err
}

func lowerPrototype(mod llvm.Module, targetData llvm.TargetData) ([]loweredState, error) {
	layouts, err := layoutFrames(mod, targetData)
	if err != nil {
		return nil, err
	}
	for _, layout := range layouts {
		if layout.plan.function.IsDeclaration() || !needsStateMachine(layout) {
			continue
		}
		if err := validateStateLayout(layout); err != nil {
			return nil, fmt.Errorf("%s: %w", layout.plan.function.Name(), err)
		}
	}
	abi := newResumeABI(mod.Context(), targetData)
	if _, err := emitLeafEntriesForLayouts(mod, abi, layouts); err != nil {
		return nil, err
	}
	var lowered []loweredState
	for _, layout := range layouts {
		if layout.plan.function.IsDeclaration() || !needsStateMachine(layout) {
			continue
		}
		entry, descriptor, err := abi.defineEntryAndDescriptor(mod, layout)
		if err != nil {
			return nil, err
		}
		lowered = append(lowered, loweredState{
			layout: layout, entry: entry, descriptor: descriptor,
		})
	}
	if err := emitStartEntriesForLayouts(mod, abi, layouts); err != nil {
		return nil, err
	}
	for i := range lowered {
		if err := lowerStateMachine(mod, targetData, abi, &lowered[i]); err != nil {
			return nil, err
		}
		if err := emitCompatibilityWrapper(mod, targetData, abi, &lowered[i]); err != nil {
			return nil, err
		}
	}
	return lowered, nil
}

func needsStateMachine(layout frameLayout) bool {
	return len(layout.plan.calls) != 0 || layout.plan.unwindSlot != 0
}

func validateStateLayout(layout frameLayout) error {
	for _, site := range layout.plan.calls {
		call := site.call
		if call.CalledFunctionType().IsFunctionVarArg() {
			return fmt.Errorf("resume call %d is variadic", site.id)
		}
		if llvm.NextInstruction(call).IsNil() {
			return fmt.Errorf("resume call %d has no continuation", site.id)
		}
	}
	return nil
}

func lowerStateMachine(
	mod llvm.Module, targetData llvm.TargetData, abi resumeABI, lowered *loweredState,
) error {
	layout := lowered.layout
	fn := layout.plan.function
	ctx := mod.Context()

	var blocks []llvm.BasicBlock
	var returns []llvm.Value
	for block := fn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		blocks = append(blocks, block)
		for instr := block.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
			if !instr.IsAReturnInst().IsNil() {
				returns = append(returns, instr)
			}
		}
	}
	if len(blocks) == 0 {
		return fmt.Errorf("%s: resumable definition has no body", fn.Name())
	}
	originalEntry := blocks[0]
	blockAddresses := collectMovedBlockAddresses(fn, blocks)

	dispatch := ctx.AddBasicBlock(lowered.entry, "dispatch")
	for _, block := range blocks {
		block.RemoveFromParent()
		llvm.AppendExistingBasicBlock(lowered.entry, block)
	}
	remapMovedBlockAddresses(lowered.entry, blockAddresses)

	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(dispatch)
	rawFrame := lowered.entry.Param(1)
	fields := make(map[uint32]llvm.Value, len(layout.plan.slots))
	for _, slot := range layout.plan.slots {
		fields[slot.id] = builder.CreateStructGEP(
			layout.typ, rawFrame, layout.fieldIndex(slot.id), "",
		)
	}

	for _, slot := range layout.plan.slots {
		if slot.kind == slotUnwind {
			continue
		}
		if isStackSave(slot.value) {
			if err := lowerPersistentStackSave(slot.value); err != nil {
				return fmt.Errorf("%s: %w", fn.Name(), err)
			}
			continue
		}
		if slot.kind == slotAlloca && slot.dynamic {
			if err := lowerDynamicAlloca(
				mod, targetData, abi, lowered.entry, slot.value, fields[slot.id],
			); err != nil {
				return fmt.Errorf("%s: %w", fn.Name(), err)
			}
			continue
		}
		switch slot.kind {
		case slotFunctionResult:
			continue
		case slotValue:
			if slot.value.InstructionOpcode() == llvm.Call &&
				isResumeCallResult(layout.plan, slot.id) {
				continue
			}
		}
		if err := spillValue(ctx, slot.value, fields[slot.id]); err != nil {
			return fmt.Errorf("%s: %w", fn.Name(), err)
		}
	}
	if err := lowerUnwindMarkers(
		ctx, lowered.entry, layout.plan, fields[layout.plan.unwindSlot],
	); err != nil {
		return fmt.Errorf("%s: %w", fn.Name(), err)
	}

	continuations := make(map[uint32]llvm.BasicBlock, len(layout.plan.calls))
	for _, site := range layout.plan.calls {
		continuation, err := splitBlockAfter(ctx, site.call, fmt.Sprintf("resume.%d", site.id))
		if err != nil {
			return fmt.Errorf("%s: call %d: %w", fn.Name(), site.id, err)
		}
		continuations[site.id] = continuation
		if site.call.CalledValue().Name() == SuspendSymbol {
			lowerSuspendCall(ctx, layout, lowered.entry, site)
			continue
		}
		if err := lowerResumeCall(
			mod, abi, layout, fields, lowered.entry, site, continuation,
		); err != nil {
			return fmt.Errorf("%s: call %d: %w", fn.Name(), site.id, err)
		}
	}

	for _, ret := range returns {
		builder.SetInsertPointBefore(ret)
		if layout.plan.resultSlot != 0 {
			builder.CreateStore(ret.Operand(0), fields[layout.plan.resultSlot])
		}
		next := builder.CreateRet(llvm.ConstInt(ctx.Int8Type(), actionReturn, false))
		next.InstructionSetDebugLoc(ret.InstructionDebugLoc())
		ret.EraseFromParentAsInstruction()
	}

	invalid := ctx.AddBasicBlock(lowered.entry, "invalid-pc")
	builder.SetInsertPointAtEnd(invalid)
	builder.CreateUnreachable()
	builder.SetInsertPointAtEnd(dispatch)
	pcField := builder.CreateStructGEP(layout.typ, rawFrame, 2, "")
	pc := builder.CreateLoad(ctx.Int32Type(), pcField, "pc")
	switchPC := builder.CreateSwitch(pc, invalid, len(continuations)+1)
	switchPC.AddCase(llvm.ConstInt(ctx.Int32Type(), 0, false), originalEntry)
	for _, site := range layout.plan.calls {
		switchPC.AddCase(
			llvm.ConstInt(ctx.Int32Type(), uint64(site.id), false),
			continuations[site.id],
		)
	}
	if layout.plan.unwindPC != 0 {
		switchPC.AddCase(
			llvm.ConstInt(ctx.Int32Type(), uint64(layout.plan.unwindPC), false),
			layout.plan.unwindBlock,
		)
	}
	return nil
}

func isResumeCallResult(plan framePlan, slotID uint32) bool {
	for _, site := range plan.calls {
		if site.resultSlot == slotID {
			return true
		}
	}
	return false
}

func lowerSuspendCall(
	ctx llvm.Context,
	parentLayout frameLayout,
	entry llvm.Value,
	site callSite,
) {
	call := site.call
	callBlock := call.InstructionParent()
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointBefore(call)
	builder.CreateStore(
		llvm.ConstInt(ctx.Int32Type(), uint64(site.id), false),
		builder.CreateStructGEP(parentLayout.typ, entry.Param(1), 2, ""),
	)
	call.EraseFromParentAsInstruction()
	terminator := callBlock.LastInstruction()
	builder.SetInsertPointBefore(terminator)
	builder.CreateRet(llvm.ConstInt(ctx.Int8Type(), actionSuspend, false))
	terminator.EraseFromParentAsInstruction()
}

func lowerResumeCall(
	mod llvm.Module,
	abi resumeABI,
	parentLayout frameLayout,
	parentFields map[uint32]llvm.Value,
	entry llvm.Value,
	site callSite,
	continuation llvm.BasicBlock,
) error {
	ctx := mod.Context()
	call := site.call
	callBlock := call.InstructionParent()
	callee := call.CalledValue()
	free := declareFrameFree(mod, abi)
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointBefore(call)

	childType := callFramePrefix(ctx, call.CalledFunctionType())
	var child llvm.Value
	if callee.IsAFunction().IsNil() || strings.HasPrefix(callee.Name(), startEntryPrefix) {
		params := append([]llvm.Type{abi.ptr}, call.CalledFunctionType().ParamTypes()...)
		startType := llvm.FunctionType(abi.ptr, params, false)
		args := make([]llvm.Value, call.CalledFunctionType().ParamTypesCount()+1)
		args[0] = entry.Param(0)
		for i := 1; i < len(args); i++ {
			args[i] = call.Operand(i - 1)
		}
		child = builder.CreateCall(startType, callee, args, "child")
	} else {
		alloc := declareFrameAllocator(mod, abi)
		descriptor := mod.NamedGlobal(descriptorPrefix + callee.Name())
		if descriptor.IsNil() {
			descriptor = llvm.AddGlobal(mod, abi.descriptorType, descriptorPrefix+callee.Name())
		}
		sizeField := builder.CreateStructGEP(abi.descriptorType, descriptor, 1, "")
		alignField := builder.CreateStructGEP(abi.descriptorType, descriptor, 2, "")
		size := builder.CreateLoad(abi.uintptrType, sizeField, "child.size")
		align := builder.CreateLoad(abi.uintptrType, alignField, "child.align")
		child = builder.CreateCall(
			alloc.GlobalValueType(), alloc, []llvm.Value{entry.Param(0), size, align}, "child",
		)
		builder.CreateIntrinsic(ctx.VoidType(), llvm.LookupIntrinsicID("llvm.memset"), []llvm.Value{
			child,
			llvm.ConstInt(ctx.Int8Type(), 0, false),
			size,
			llvm.ConstInt(ctx.Int1Type(), 0, false),
		}, "")

		builder.CreateStore(entry.Param(1), builder.CreateStructGEP(childType, child, 0, ""))
		builder.CreateStore(descriptor, builder.CreateStructGEP(childType, child, 1, ""))
		builder.CreateStore(
			llvm.ConstInt(ctx.Int32Type(), 0, false),
			builder.CreateStructGEP(childType, child, 2, ""),
		)
		for i := 0; i < call.CalledFunctionType().ParamTypesCount(); i++ {
			builder.CreateStore(
				call.Operand(i),
				builder.CreateStructGEP(childType, child, frameHeaderFields+i, ""),
			)
		}
	}
	builder.CreateStore(
		llvm.ConstInt(ctx.Int32Type(), uint64(site.id), false),
		builder.CreateStructGEP(parentLayout.typ, entry.Param(1), 2, ""),
	)
	contextTop := builder.CreateStructGEP(abi.contextType, entry.Param(0), 0, "")
	builder.CreateStore(child, contextTop)

	builder.SetInsertPointBefore(continuation.FirstInstruction())
	returnedField := builder.CreateStructGEP(abi.contextType, entry.Param(0), 1, "")
	returned := builder.CreateLoad(abi.ptr, returnedField, "returned")
	builder.CreateStore(llvm.ConstNull(abi.ptr), returnedField)
	if site.resultSlot != 0 {
		resultField := frameHeaderFields + call.CalledFunctionType().ParamTypesCount()
		result := builder.CreateLoad(
			call.Type(), builder.CreateStructGEP(childType, returned, resultField, ""), "call.result",
		)
		builder.CreateStore(result, parentFields[site.resultSlot])
		replaceValueUsesWithLoads(ctx, call, parentFields[site.resultSlot], llvm.Value{})
	}
	builder.CreateCall(
		free.GlobalValueType(), free, []llvm.Value{entry.Param(0), returned}, "",
	)

	call.EraseFromParentAsInstruction()
	terminator := callBlock.LastInstruction()
	builder.SetInsertPointBefore(terminator)
	builder.CreateRet(llvm.ConstInt(ctx.Int8Type(), actionContinue, false))
	terminator.EraseFromParentAsInstruction()
	return nil
}

func callFramePrefix(ctx llvm.Context, typ llvm.Type) llvm.Type {
	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	fields := []llvm.Type{ptr, ptr, ctx.Int32Type()}
	fields = append(fields, typ.ParamTypes()...)
	if result := typ.ReturnType(); result.TypeKind() != llvm.VoidTypeKind {
		fields = append(fields, result)
	}
	return ctx.StructType(fields, false)
}

func declareFrameAllocator(mod llvm.Module, abi resumeABI) llvm.Value {
	fn := mod.NamedFunction(frameAllocName)
	if fn.IsNil() {
		fn = llvm.AddFunction(mod, frameAllocName, llvm.FunctionType(
			abi.ptr, []llvm.Type{abi.ptr, abi.uintptrType, abi.uintptrType}, false,
		))
	}
	return fn
}

func declareFrameFree(mod llvm.Module, abi resumeABI) llvm.Value {
	fn := mod.NamedFunction(frameFreeName)
	if fn.IsNil() {
		fn = llvm.AddFunction(mod, frameFreeName, llvm.FunctionType(
			abi.ctx.VoidType(), []llvm.Type{abi.ptr, abi.ptr}, false,
		))
	}
	return fn
}

func declareFrameClose(mod llvm.Module, abi resumeABI) llvm.Value {
	fn := mod.NamedFunction(frameCloseName)
	if fn.IsNil() {
		fn = llvm.AddFunction(mod, frameCloseName, llvm.FunctionType(
			abi.ctx.VoidType(), []llvm.Type{abi.ptr}, false,
		))
	}
	return fn
}
