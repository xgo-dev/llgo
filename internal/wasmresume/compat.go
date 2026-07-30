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

	"github.com/xgo-dev/llvm"
)

// emitCompatibilityWrapper keeps the original Go symbol callable from
// non-resumable runtime and C boundaries. Such a call owns a temporary context
// and must run to completion; observing Suspend is a boundary violation.
func emitCompatibilityWrapper(
	mod llvm.Module, targetData llvm.TargetData, abi resumeABI, lowered *loweredState,
) error {
	fn := lowered.layout.plan.function
	if !fn.IsDeclaration() {
		return fmt.Errorf("%s: compatibility wrapper still has a body", fn.Name())
	}

	ctx := mod.Context()
	entry := ctx.AddBasicBlock(fn, "wasm.resume.compat")
	dispatch := ctx.AddBasicBlock(fn, "wasm.resume.dispatch")
	resume := ctx.AddBasicBlock(fn, "wasm.resume.call")
	continued := ctx.AddBasicBlock(fn, "wasm.resume.continue")
	returned := ctx.AddBasicBlock(fn, "wasm.resume.return")
	finished := ctx.AddBasicBlock(fn, "wasm.resume.finished")
	suspended := ctx.AddBasicBlock(fn, "wasm.resume.suspended")
	invalid := ctx.AddBasicBlock(fn, "wasm.resume.invalid")

	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(entry)
	context := builder.CreateAlloca(abi.contextType, "resume.context")
	context.SetAlignment(targetData.ABITypeAlignment(abi.contextType))
	root := builder.CreateAlloca(lowered.layout.typ, "resume.root")
	root.SetAlignment(lowered.layout.alignment)
	builder.CreateIntrinsic(ctx.VoidType(), llvm.LookupIntrinsicID("llvm.memset"), []llvm.Value{
		context,
		llvm.ConstInt(ctx.Int8Type(), 0, false),
		llvm.ConstInt(abi.uintptrType, targetData.TypeAllocSize(abi.contextType), false),
		llvm.ConstInt(ctx.Int1Type(), 0, false),
	}, "")
	builder.CreateIntrinsic(ctx.VoidType(), llvm.LookupIntrinsicID("llvm.memset"), []llvm.Value{
		root,
		llvm.ConstInt(ctx.Int8Type(), 0, false),
		llvm.ConstInt(abi.uintptrType, lowered.layout.size, false),
		llvm.ConstInt(ctx.Int1Type(), 0, false),
	}, "")
	builder.CreateStore(
		llvm.ConstNull(abi.ptr),
		builder.CreateStructGEP(lowered.layout.typ, root, 0, ""),
	)
	builder.CreateStore(
		lowered.descriptor,
		builder.CreateStructGEP(lowered.layout.typ, root, 1, ""),
	)
	for _, slot := range lowered.layout.plan.slots {
		if slot.kind != slotParameter {
			continue
		}
		builder.CreateStore(
			fn.Param(parameterIndex(lowered.layout.plan, slot.id)),
			builder.CreateStructGEP(
				lowered.layout.typ, root, lowered.layout.fieldIndex(slot.id), "",
			),
		)
	}
	topField := builder.CreateStructGEP(abi.contextType, context, 0, "")
	returnedField := builder.CreateStructGEP(abi.contextType, context, 1, "")
	builder.CreateStore(root, topField)
	builder.CreateBr(dispatch)

	framePrefix := ctx.StructType([]llvm.Type{abi.ptr, abi.ptr, ctx.Int32Type()}, false)
	builder.SetInsertPointAtEnd(dispatch)
	top := builder.CreateLoad(abi.ptr, topField, "top")
	builder.CreateCondBr(
		builder.CreateICmp(llvm.IntNE, top, llvm.ConstNull(abi.ptr), ""),
		resume,
		finished,
	)

	builder.SetInsertPointAtEnd(resume)
	descriptor := builder.CreateLoad(
		abi.ptr, builder.CreateStructGEP(framePrefix, top, 1, ""), "descriptor",
	)
	resumeEntry := builder.CreateLoad(
		abi.ptr,
		builder.CreateStructGEP(abi.descriptorType, descriptor, 0, ""),
		"resume.entry",
	)
	action := builder.CreateCall(abi.entryType, resumeEntry, []llvm.Value{context, top}, "action")
	actionSwitch := builder.CreateSwitch(action, invalid, 3)
	actionSwitch.AddCase(llvm.ConstInt(ctx.Int8Type(), actionContinue, false), continued)
	actionSwitch.AddCase(llvm.ConstInt(ctx.Int8Type(), actionReturn, false), returned)
	actionSwitch.AddCase(llvm.ConstInt(ctx.Int8Type(), actionSuspend, false), suspended)

	builder.SetInsertPointAtEnd(continued)
	builder.CreateBr(dispatch)

	builder.SetInsertPointAtEnd(returned)
	parent := builder.CreateLoad(
		abi.ptr, builder.CreateStructGEP(framePrefix, top, 0, ""), "parent",
	)
	builder.CreateStore(parent, topField)
	builder.CreateStore(top, returnedField)
	builder.CreateBr(dispatch)

	builder.SetInsertPointAtEnd(finished)
	builder.CreateCall(
		declareFrameClose(mod, abi).GlobalValueType(),
		declareFrameClose(mod, abi),
		[]llvm.Value{context},
		"",
	)
	if lowered.layout.plan.resultSlot == 0 {
		builder.CreateRetVoid()
	} else {
		result := builder.CreateLoad(
			fn.GlobalValueType().ReturnType(),
			builder.CreateStructGEP(
				lowered.layout.typ,
				root,
				lowered.layout.fieldIndex(lowered.layout.plan.resultSlot),
				"",
			),
			"result",
		)
		builder.CreateRet(result)
	}

	builder.SetInsertPointAtEnd(suspended)
	builder.CreateIntrinsic(ctx.VoidType(), llvm.LookupIntrinsicID("llvm.trap"), nil, "")
	builder.CreateUnreachable()
	builder.SetInsertPointAtEnd(invalid)
	builder.CreateUnreachable()
	return nil
}

func parameterIndex(plan framePlan, slotID uint32) int {
	index := 0
	for _, slot := range plan.slots {
		if slot.kind != slotParameter {
			continue
		}
		if slot.id == slotID {
			return index
		}
		index++
	}
	return -1
}
