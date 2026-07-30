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

const (
	resumeEntryPrefix = "__llgo_wasm_resume."
	startEntryPrefix  = "__llgo_wasm_start."
	descriptorPrefix  = "__llgo_wasm_resume_desc."
	frameCloseName    = "__llgo_wasm_resume_close"
	actionContinue    = 0
	actionReturn      = 1
	actionSuspend     = 2
)

// StartSymbol returns the resumable start entry for a Go function symbol.
func StartSymbol(function string) string {
	return startEntryPrefix + function
}

type resumeABI struct {
	ctx            llvm.Context
	ptr            llvm.Type
	uintptrType    llvm.Type
	entryType      llvm.Type
	descriptorType llvm.Type
	contextType    llvm.Type
}

func newResumeABI(ctx llvm.Context, targetData llvm.TargetData) resumeABI {
	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	uintptrType := ctx.IntType(targetData.PointerSize() * 8)
	return resumeABI{
		ctx:         ctx,
		ptr:         ptr,
		uintptrType: uintptrType,
		entryType:   llvm.FunctionType(ctx.Int8Type(), []llvm.Type{ptr, ptr}, false),
		descriptorType: ctx.StructType([]llvm.Type{
			ptr,
			uintptrType,
			uintptrType,
			uintptrType,
			ctx.Int32Type(),
		}, false),
		// The first two fields are the public dispatch ABI. The trailing pointer
		// is runtime-owned per-context frame storage.
		contextType: ctx.StructType([]llvm.Type{ptr, ptr, ptr}, false),
	}
}

func (abi resumeABI) defineEntryAndDescriptor(
	mod llvm.Module, layout frameLayout,
) (entry, descriptor llvm.Value, err error) {
	fn := layout.plan.function
	entryName := resumeEntryPrefix + fn.Name()
	descriptorName := descriptorPrefix + fn.Name()
	if !mod.NamedFunction(entryName).IsNil() || !mod.NamedGlobal(descriptorName).IsNil() {
		return llvm.Value{}, llvm.Value{}, fmt.Errorf("%s: duplicate resumable descriptor", fn.Name())
	}

	entry = llvm.AddFunction(mod, entryName, abi.entryType)
	entry.SetLinkage(llvm.InternalLinkage)
	descriptor = llvm.AddGlobal(mod, abi.descriptorType, descriptorName)
	descriptor.SetLinkage(fn.Linkage())
	descriptor.SetGlobalConstant(true)
	descriptor.SetInitializer(abi.ctx.ConstStruct([]llvm.Value{
		entry,
		llvm.ConstInt(abi.uintptrType, layout.size, false),
		llvm.ConstInt(abi.uintptrType, uint64(layout.alignment), false),
		llvm.ConstInt(abi.uintptrType, layout.unwindOffset, false),
		llvm.ConstInt(abi.ctx.Int32Type(), uint64(layout.plan.unwindPC), false),
	}, false))
	return entry, descriptor, nil
}
