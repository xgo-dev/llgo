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

package ssa

import (
	"go/types"

	llabi "github.com/xgo-dev/llgo/internal/abi"
	"github.com/xgo-dev/llvm"
)

// GoWordSize is the storage width of int, uint, uintptr, and Go pointers. The
// official Go WebAssembly ports use an eight-byte language word even when Core
// Wasm addresses are i32. Pointer expressions remain LLVM pointers so memory
// operations use the physical address width.
func (p Program) GoWordSize() int {
	return p.goWordSize
}

func (p Program) usesWideGoStorage() bool {
	return p.goWordSize > p.ptrSize
}

// Go's widened data slot and C's native pointer followed by padding occupy
// the same bytes in GoString and GoSlice. Keep their LLVM prototypes identical
// so a noncapturing Go callback can retain its code address.
func (p Program) isWasmABIPaddedHeader(t llvm.Type) bool {
	return t == p.rtStringTy || t == p.rtSliceTy
}

func (p Program) hasWasmABIPadding(t llvm.Type) bool {
	return p.usesWideGoStorage() && p.wasmHeaderCount(t, 1) != 0
}

// wasmHeaderCount counts string and slice headers up to limit. Inspect each
// array element type once, irrespective of the array length.
func (p Program) wasmHeaderCount(t llvm.Type, limit int) int {
	if p.isWasmABIPaddedHeader(t) {
		return 1
	}
	switch t.TypeKind() {
	case llvm.ArrayTypeKind:
		n := min(t.ArrayLength(), limit)
		if n != 0 {
			return min(limit, n*p.wasmHeaderCount(t.ElementType(), limit))
		}
	case llvm.StructTypeKind:
		count := 0
		for _, field := range t.StructElementTypes() {
			count += p.wasmHeaderCount(field, limit-count)
			if count == limit {
				break
			}
		}
		return count
	}
	return 0
}

// clearWasmABIPadding zeroes the high word of each string or slice data slot.
func (p Program) clearWasmABIPadding(b llvm.Builder, value llvm.Value) llvm.Value {
	t := value.Type()
	if p.isWasmABIPaddedHeader(t) {
		slot := b.CreateExtractValue(value, 0, "")
		highWord := slot.Type().StructElementTypes()[1]
		slot = b.CreateInsertValue(slot, llvm.ConstNull(highWord), 1, "")
		return b.CreateInsertValue(value, slot, 0, "")
	}
	result := value
	update := func(i int) {
		field := b.CreateExtractValue(value, i, "")
		result = b.CreateInsertValue(result, p.clearWasmABIPadding(b, field), i, "")
	}
	if t.TypeKind() == llvm.ArrayTypeKind {
		for i := 0; i < t.ArrayLength(); i++ {
			update(i)
		}
	} else {
		for i, field := range t.StructElementTypes() {
			if p.hasWasmABIPadding(field) {
				update(i)
			}
		}
	}
	return result
}

func (f Function) finishWasmSliceParams() {
	if !f.Prog.usesWideGoStorage() || f.env != nil {
		return
	}
	// A C callback can call any environment-free Go entry. Rewrite each used
	// parameter once, including values obtained before the body was built.
	for i := range f.params {
		param := f.impl.Param(i)
		if param.FirstUse().IsNil() || !f.Prog.hasWasmSlicePadding(param.Type()) {
			continue
		}
		b := f.entryAllocaBuilder()
		placeholder := b.CreatePHI(param.Type(), "")
		param.ReplaceAllUsesWith(placeholder)
		value := f.Prog.clearWasmABIPadding(b, param)
		placeholder.ReplaceAllUsesWith(value)
		placeholder.EraseFromParentAsInstruction()
	}
}

func (p Program) needsWidePointerStorage(t Type) bool {
	if !p.usesWideGoStorage() || p.isNativeStorage(t) {
		return false
	}
	switch t.kind {
	case vkPtr, vkFuncPtr, vkMap, vkChan:
		return true
	default:
		return false
	}
}

func (p Program) isNativeStorage(t Type) bool {
	return t != nil && p.nativeStorage[t] == t
}

func (p Program) withNativeStorage(t Type) Type {
	if t == nil || !p.usesWideGoStorage() {
		return t
	}
	if view := p.nativeStorage[t]; view != nil {
		return view
	}
	view := *t
	view.ll = p.nativeStorageLLVMType(t.raw.Type, t.ll)
	p.nativeStorage[t] = &view
	p.nativeStorage[&view] = &view
	return &view
}

func (p Program) nativeStorageLLVMType(raw types.Type, fallback llvm.Type) llvm.Type {
	if !p.usesWideGoStorage() {
		return fallback
	}
	switch t := types.Unalias(raw).Underlying().(type) {
	case *types.Basic:
		if t.Kind() == types.Uintptr {
			return llvmIntType(p.ctx, p.PointerSize())
		}
	case *types.Array:
		elem := p.rawType(t.Elem())
		return llvm.ArrayType(p.nativeStorageLLVMType(t.Elem(), elem.ll), int(t.Len()))
	}
	return fallback
}

func (b Builder) logicalValue(v Expr) Expr {
	t := b.Prog.withoutNativeStorage(v.Type)
	if t == v.Type {
		return v
	}
	if v.impl.Type() != t.ll {
		v.impl = castInt(b, v.impl, v.Type, t)
	}
	v.Type = t
	return v
}

func (b Builder) fitLLVMValue(value llvm.Value, source Type, target llvm.Type) llvm.Value {
	if value.Type() == target {
		return value
	}
	if source.kind == vkArray && value.Type().TypeKind() == llvm.ArrayTypeKind && target.TypeKind() == llvm.ArrayTypeKind && value.Type().ArrayLength() == target.ArrayLength() {
		elem := b.Prog.Index(source)
		result := llvm.Undef(target)
		for i, n := 0, target.ArrayLength(); i < n; i++ {
			part := b.impl.CreateExtractValue(value, i, "")
			part = b.fitLLVMValue(part, elem, target.ElementType())
			result = b.impl.CreateInsertValue(result, part, i, "")
		}
		return result
	}
	logical := b.Prog.withoutNativeStorage(source)
	if logical.kind == vkSIMD {
		storage := b.Prog.storageType(logical)
		if value.Type() == logical.ll && target == storage {
			return b.toStorageValue(logical, value)
		}
		if value.Type() == storage && target == logical.ll {
			return b.fromStorageValue(logical, value)
		}
	}
	if b.Prog.needsWidePointerStorage(logical) {
		wide := b.Prog.widePointerStorageType()
		if value.Type() == wide && target.TypeKind() == llvm.PointerTypeKind {
			return b.fromStorageValue(logical, value)
		}
		if value.Type().TypeKind() == llvm.PointerTypeKind && target == wide {
			return b.toStorageValue(logical, value)
		}
	}
	if value.Type().TypeKind() == llvm.IntegerTypeKind && target.TypeKind() == llvm.IntegerTypeKind {
		if value.Type().IntTypeWidth() > target.IntTypeWidth() {
			truncated := llvm.CreateTrunc(b.impl, value, target)
			var restored llvm.Value
			if source.kind == vkUnsigned {
				restored = llvm.CreateZExt(b.impl, truncated, value.Type())
			} else {
				restored = llvm.CreateSExt(b.impl, truncated, value.Type())
			}
			overflow := llvm.CreateICmp(b.impl, llvm.IntNE, restored, value)
			b.assertRuntimeError(overflow, "WebAssembly ABI integer conversion out of range")
			return truncated
		}
		t := *source
		t.ll = target
		return castInt(b, value, source, &t)
	}
	return value
}

func (b Builder) fitLLVMResult(value llvm.Value, logical Type) llvm.Value {
	if value.Type() == logical.ll {
		return value
	}
	if logical.kind != vkTuple || value.Type().TypeKind() != llvm.StructTypeKind {
		return b.fitLLVMValue(value, logical, logical.ll)
	}
	fields := value.Type().StructElementTypes()
	values := make([]llvm.Value, len(fields))
	for i := range fields {
		field := b.Prog.Field(logical, i)
		part := b.impl.CreateExtractValue(value, i, "")
		values[i] = b.fitLLVMValue(part, field, field.ll)
	}
	return b.aggregateValue(logical, values...).impl
}

func (p Program) withoutNativeStorage(t Type) Type {
	if p.isNativeStorage(t) {
		return p.rawType(t.raw.Type)
	}
	return t
}

func (p Program) childStorageType(parent Type, child types.Type) Type {
	t := p.rawType(child)
	if !p.usesWideGoStorage() {
		return t
	}
	// A record's declaration chooses the width of its fields. Referencing an
	// ordinary Go record from C still uses the layout built by toLLVMFields.
	native := p.isNativeStorage(parent)
	switch parent.kind {
	case vkStruct:
		native = p.hasLegacyNativeTypeLayout(parent.raw.Type)
	case vkArray:
		// Named native arrays keep their physical element type even when
		// reached through an ordinary Go record or pointer.
		native = parent.ll.ElementType() != p.storageType(t)
	}
	if native {
		return p.withNativeStorage(t)
	}
	return t
}

func (p Program) aggregateElementType(parent Type, index int) Type {
	if _, ok := types.Unalias(parent.raw.Type).Underlying().(*types.Array); ok {
		return p.Index(parent)
	}
	return p.Field(parent, index)
}

func (p Program) storageType(t Type) llvm.Type {
	if t.kind == vkSIMD {
		return p.rawType(t.RawType().Underlying()).ll
	}
	if p.needsWidePointerStorage(t) {
		return p.widePointerStorageType()
	}
	return t.ll
}

func (p Program) requireStorageAlignment(value llvm.Value, t Type) {
	if !p.usesWideGoStorage() {
		return
	}
	storage := p.storageType(t)
	want := p.AlignOf(t)
	if have := uint64(p.td.ABITypeAlignment(storage)); want > have {
		value.SetAlignment(int(want))
	}
}

// widePointerStorageType preserves a native wasm32 pointer relocation in the
// low word and zero-fills the high word. The allocation or enclosing Go
// aggregate supplies the required eight-byte alignment. LLVM's
// WebAssembly backend cannot lower ptrtoint constant expressions as static
// relocations, so representing this slot directly as i64 is not viable.
func (p Program) widePointerStorageType() llvm.Type {
	if p.widePtrStorageTy.IsNil() {
		p.widePtrStorageTy = p.ctx.StructType([]llvm.Type{
			p.tyVoidPtr(),
			llvmIntType(p.ctx, p.GoWordSize()-p.PointerSize()),
		}, false)
	}
	return p.widePtrStorageTy
}

func (b Builder) toStorageValue(t Type, value llvm.Value) llvm.Value {
	if t.kind == vkSIMD {
		return b.simdToStorage(value, t)
	}
	if b.Prog.needsWidePointerStorage(t) {
		storage := b.Prog.widePointerStorageType()
		return b.impl.CreateInsertValue(llvm.ConstNull(storage), value, 0, "")
	}
	return b.fitLLVMValue(value, b.Prog.withoutNativeStorage(t), t.ll)
}

// A Go pointer or slice can alias C memory after it has passed through
// another package. A typed load canonicalizes its value without writing back
// into the source or depending on the pointer retaining its storage view.
func (b Builder) loadStorageValue(ptr llvm.Value, storage, logical Type) llvm.Value {
	count := 0
	var scratch llvm.Value
	if b.Prog.usesWideGoStorage() {
		count = b.Prog.wasmHeaderCount(logical.ll, maxInlineWasmHeaders+1)
		if count > maxInlineWasmHeaders {
			scratch = b.wasmHeaderSnapshotStorage(logical)
		}
	}

	stored := llvm.CreateLoad(b.impl, b.Prog.storageType(storage), ptr)
	b.setGoAccessAlignment(stored, storage)
	value := b.fromStorageValue(storage, stored)
	switch {
	case count == 0:
		return value
	case count <= maxInlineWasmHeaders:
		return b.Prog.clearWasmABIPadding(b.impl, value)
	default:
		b.impl.CreateStore(value, scratch)
		b.clearWasmHeaderRange(logical.ll, scratch, b.Prog.IntVal(1, b.Prog.Int()).impl)
		result := llvm.CreateLoad(b.impl, logical.ll, scratch)
		ctx := b.Prog.ctx
		result.SetMetadata(ctx.MDKindID(llabi.HeaderSnapshotMD), ctx.MDNode(nil))
		return result
	}
}

func (b Builder) wasmHeaderSnapshotStorage(typ Type) llvm.Value {
	size := b.Prog.SizeOf(typ)
	if size >= llabi.MinAggregateCopySize {
		// ABI lowering can use this private scanned snapshot for delayed
		// consumers or forward an immediate copy to its destination.
		// Allocate before reading the original while its owner is live.
		return b.AllocU(typ).impl
	}
	// A snapshot lives only through the range helper's scalar stores and its
	// immediately following SSA load. All sites in this invocation can share
	// a single slot. Recursion has its own frame and there is no safepoint.
	slot := b.Func.wasmHeaderScratch
	if slot.IsNil() {
		slot = b.Func.entryAllocaBuilder().CreateArrayAlloca(b.Prog.tyInt8(), llvm.ConstInt(b.Prog.tyInt32(), size, false), "")
		b.Func.wasmHeaderScratch = slot
	} else if size > slot.Operand(0).ZExtValue() {
		slot.SetOperand(0, llvm.ConstInt(b.Prog.tyInt32(), size, false))
	}
	if align := int(b.Prog.AlignOf(typ)); align > slot.Alignment() {
		slot.SetAlignment(align)
	}
	return slot
}

func (b Builder) fromStorageValue(t Type, value llvm.Value) llvm.Value {
	logical := b.Prog.withoutNativeStorage(t)
	if value.Type() == logical.ll {
		return value
	}
	if t.kind == vkSIMD {
		return b.simdFromStorage(value, t)
	}
	if b.Prog.needsWidePointerStorage(t) {
		return b.impl.CreateExtractValue(value, 0, "")
	}
	return b.fitLLVMValue(value, t, logical.ll)
}

func (p Program) toStorageConstant(t Type, value llvm.Value) llvm.Value {
	if t.kind == vkSIMD {
		lanes := simdLanes(t.RawType())
		values := make([]llvm.Value, lanes.Len())
		for i := range values {
			values[i] = llvm.ConstExtractElement(value, llvm.ConstInt(p.tyInt32(), uint64(i), false))
		}
		storage := p.rawType(t.RawType().Underlying())
		return p.constStructValue(storage, []llvm.Value{p.Zero(p.Field(storage, 0)).impl, llvm.ConstArray(p.rawType(lanes.Elem()).ll, values)})
	}
	if p.needsWidePointerStorage(t) {
		storage := p.widePointerStorageType()
		parts := storage.StructElementTypes()
		return p.ctx.ConstStruct([]llvm.Value{
			value,
			llvm.ConstNull(parts[1]),
		}, false)
	}
	return p.fitLLVMConstant(value, t, t.ll)
}

func (p Program) fitLLVMConstant(value llvm.Value, source Type, target llvm.Type) llvm.Value {
	if value.Type() == target || value.Type().TypeKind() != llvm.IntegerTypeKind || target.TypeKind() != llvm.IntegerTypeKind {
		return value
	}
	fromBits := value.Type().IntTypeWidth()
	toBits := target.IntTypeWidth()
	if fromBits > toBits {
		if integer := value.IsAConstantInt(); !integer.IsNil() {
			if source.kind == vkUnsigned {
				if toBits < 64 && integer.ZExtValue() > (uint64(1)<<toBits)-1 {
					panic("ssa: WebAssembly ABI integer constant out of range")
				}
			} else if toBits < 64 {
				v := integer.SExtValue()
				limit := int64(1) << (toBits - 1)
				if v < -limit || v >= limit {
					panic("ssa: WebAssembly ABI integer constant out of range")
				}
			}
		}
		return llvm.ConstTrunc(value, target)
	}
	integer := value.IsAConstantInt()
	if integer.IsNil() {
		panic("ssa: unsupported WebAssembly ABI integer constant extension")
	}
	if source.kind == vkUnsigned {
		return llvm.ConstInt(target, integer.ZExtValue(), false)
	}
	return llvm.ConstInt(target, uint64(integer.SExtValue()), true)
}

func (b Builder) toAtomicStorageValue(t Type, value llvm.Value) llvm.Value {
	if b.Prog.needsWidePointerStorage(t) {
		physical := llvmIntType(b.Prog.ctx, b.Prog.PointerSize())
		value = llvm.CreatePtrToInt(b.impl, value, physical)
		return llvm.CreateZExt(b.impl, value, b.Prog.tyInt())
	}
	return b.toStorageValue(t, value)
}

func (b Builder) fromAtomicStorageValue(t Type, value llvm.Value) llvm.Value {
	if b.Prog.needsWidePointerStorage(t) {
		physical := llvmIntType(b.Prog.ctx, b.Prog.PointerSize())
		value = llvm.CreateTrunc(b.impl, value, physical)
		return llvm.CreateIntToPtr(b.impl, value, t.ll)
	}
	return b.fromStorageValue(t, value)
}

// physicalPointerIndex converts a semantically Go-sized index to the integer
// width used by the target's linear-memory addresses. LLVM permits wider GEP
// indices, but the WebAssembly backend can otherwise leave an i64 address on a
// memory32 load at -O0 instead of inserting i32.wrap_i64.
func (b Builder) physicalPointerIndex(index Expr) Expr {
	if !b.Prog.usesWideGoStorage() {
		return index
	}
	physical := llvmIntType(b.Prog.ctx, b.Prog.PointerSize())
	if index.impl.Type() == physical {
		return index
	}
	if index.impl.Type().TypeKind() != llvm.IntegerTypeKind {
		panic("ssa: non-integer pointer index")
	}
	if index.impl.Type().IntTypeWidth() <= physical.IntTypeWidth() {
		return index
	}
	index.impl = llvm.CreateTrunc(b.impl, index.impl, physical)
	t := *index.Type
	t.ll = physical
	index.Type = &t
	return index
}

// The legacy call adapter only converts signatures containing native slices.
func (p Program) hasWasmSlicePadding(t llvm.Type) bool {
	if !p.usesWideGoStorage() || p.rtSliceTy.IsNil() {
		return false
	}
	if t == p.rtSliceTy {
		return true
	}
	switch t.TypeKind() {
	case llvm.StructTypeKind:
		for _, field := range t.StructElementTypes() {
			if p.hasWasmSlicePadding(field) {
				return true
			}
		}
	case llvm.ArrayTypeKind:
		return t.ArrayLength() > 0 && p.hasWasmSlicePadding(t.ElementType())
	}
	return false
}
