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

func emitStartEntriesForLayouts(
	mod llvm.Module, abi resumeABI, layouts []frameLayout,
) error {
	ctx := mod.Context()
	var alloc llvm.Value
	for _, layout := range layouts {
		fn := layout.plan.function
		if fn.IsDeclaration() || fn.GlobalValueType().IsFunctionVarArg() {
			continue
		}
		descriptor := mod.NamedGlobal(descriptorPrefix + fn.Name())
		if descriptor.IsNil() || descriptor.Initializer().IsNil() {
			return fmt.Errorf("%s: resumable descriptor is not defined", fn.Name())
		}

		params := append([]llvm.Type{abi.ptr}, fn.GlobalValueType().ParamTypes()...)
		startType := llvm.FunctionType(abi.ptr, params, false)
		startName := StartSymbol(fn.Name())
		start := mod.NamedFunction(startName)
		if start.IsNil() {
			start = llvm.AddFunction(mod, startName, startType)
		} else if !start.IsDeclaration() || start.GlobalValueType() != startType {
			return fmt.Errorf("%s: incompatible resumable start entry", fn.Name())
		}
		start.SetLinkage(fn.Linkage())
		if alloc.IsNil() {
			alloc = declareFrameAllocator(mod, abi)
		}

		block := ctx.AddBasicBlock(start, "entry")
		builder := ctx.NewBuilder()
		builder.SetInsertPointAtEnd(block)
		child := builder.CreateCall(alloc.GlobalValueType(), alloc, []llvm.Value{
			start.Param(0),
			llvm.ConstInt(abi.uintptrType, layout.size, false),
			llvm.ConstInt(abi.uintptrType, uint64(layout.alignment), false),
		}, "child")
		builder.CreateIntrinsic(ctx.VoidType(), llvm.LookupIntrinsicID("llvm.memset"), []llvm.Value{
			child,
			llvm.ConstInt(ctx.Int8Type(), 0, false),
			llvm.ConstInt(abi.uintptrType, layout.size, false),
			llvm.ConstInt(ctx.Int1Type(), 0, false),
		}, "")

		contextTop := builder.CreateStructGEP(abi.contextType, start.Param(0), 0, "")
		parent := builder.CreateLoad(abi.ptr, contextTop, "parent")
		builder.CreateStore(parent, builder.CreateStructGEP(layout.typ, child, 0, ""))
		builder.CreateStore(descriptor, builder.CreateStructGEP(layout.typ, child, 1, ""))
		builder.CreateStore(
			llvm.ConstInt(ctx.Int32Type(), 0, false),
			builder.CreateStructGEP(layout.typ, child, 2, ""),
		)
		param := 1
		for _, slot := range layout.plan.slots {
			if slot.kind != slotParameter {
				continue
			}
			builder.CreateStore(
				start.Param(param),
				builder.CreateStructGEP(layout.typ, child, layout.fieldIndex(slot.id), ""),
			)
			param++
		}
		builder.CreateRet(child)
		builder.Dispose()
	}
	return nil
}
