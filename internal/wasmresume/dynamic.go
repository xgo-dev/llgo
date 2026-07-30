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

func lowerDynamicAlloca(
	mod llvm.Module,
	targetData llvm.TargetData,
	abi resumeABI,
	entry llvm.Value,
	alloca llvm.Value,
	field llvm.Value,
) error {
	if alloca.IsAAllocaInst().IsNil() || alloca.OperandsCount() == 0 {
		return fmt.Errorf("invalid dynamic alloca %q", alloca.Name())
	}

	ctx := mod.Context()
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointBefore(alloca)

	count := alloca.Operand(0)
	switch {
	case count.Type().IntTypeWidth() < abi.uintptrType.IntTypeWidth():
		count = builder.CreateZExt(count, abi.uintptrType, "alloca.count")
	case count.Type().IntTypeWidth() > abi.uintptrType.IntTypeWidth():
		count = builder.CreateTrunc(count, abi.uintptrType, "alloca.count")
	}
	size := count
	if elementSize := targetData.TypeAllocSize(alloca.AllocatedType()); elementSize != 1 {
		size = builder.CreateMul(
			count,
			llvm.ConstInt(abi.uintptrType, elementSize, false),
			"alloca.size",
		)
	}
	one := llvm.ConstInt(abi.uintptrType, 1, false)
	size = builder.CreateSelect(
		builder.CreateICmp(llvm.IntEQ, size, llvm.ConstNull(abi.uintptrType), ""),
		one,
		size,
		"alloca.nonzero.size",
	)
	align := targetData.ABITypeAlignment(alloca.AllocatedType())
	if alloca.Alignment() > align {
		align = alloca.Alignment()
	}
	allocate := declareDynamicAllocator(mod, abi)
	value := builder.CreateCall(allocate.GlobalValueType(), allocate, []llvm.Value{
		entry.Param(0),
		size,
		llvm.ConstInt(abi.uintptrType, uint64(align), false),
	}, alloca.Name()+".frame")
	store := builder.CreateStore(value, field)
	replaceValueUsesWithLoads(ctx, alloca, field, store)
	alloca.EraseFromParentAsInstruction()
	return nil
}

func isStackSave(value llvm.Value) bool {
	return isCallToIntrinsic(value, "llvm.stacksave")
}

func lowerPersistentStackSave(save llvm.Value) error {
	var restores []llvm.Value
	seen := make(map[llvm.Value]struct{})
	for use := save.FirstUse(); !use.IsNil(); use = use.NextUse() {
		user := use.User()
		if !isCallToIntrinsic(user, "llvm.stackrestore") {
			return fmt.Errorf("persistent stacksave has unsupported use %q", user.Name())
		}
		if _, ok := seen[user]; !ok {
			seen[user] = struct{}{}
			restores = append(restores, user)
		}
	}
	for _, restore := range restores {
		restore.EraseFromParentAsInstruction()
	}
	save.EraseFromParentAsInstruction()
	return nil
}

func isCallToIntrinsic(value llvm.Value, name string) bool {
	if value.IsNil() || value.IsAInstruction().IsNil() ||
		value.InstructionOpcode() != llvm.Call {
		return false
	}
	callee := value.CalledValue()
	return !callee.IsAFunction().IsNil() &&
		(callee.Name() == name || strings.HasPrefix(callee.Name(), name+"."))
}

func declareDynamicAllocator(mod llvm.Module, abi resumeABI) llvm.Value {
	fn := mod.NamedFunction(frameDynamicAllocName)
	if fn.IsNil() {
		fn = llvm.AddFunction(mod, frameDynamicAllocName, llvm.FunctionType(
			abi.ptr, []llvm.Type{abi.ptr, abi.uintptrType, abi.uintptrType}, false,
		))
	}
	return fn
}
