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

	"github.com/xgo-dev/llvm"
)

// GoWordSize is the storage width of int, uint, uintptr, and Go pointers. The
// official Go WebAssembly ports use an eight-byte language word even when Core
// Wasm addresses are i32. Pointer expressions remain LLVM pointers so memory
// operations use the physical address width.
func (p Program) GoWordSize() int {
	if p.target != nil && p.target.effectiveGOARCH() == "wasm" && p.target.WasmProfile != "" {
		return 8
	}
	return p.PointerSize()
}

func (p Program) needsWidePointerStorage(t Type) bool {
	if p.GoWordSize() <= p.PointerSize() || p.isNativeStorage(t) {
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
	_, ok := p.nativeStorage[t]
	return ok
}

func (p Program) withNativeStorage(t Type) Type {
	if t == nil || p.isNativeStorage(t) {
		return t
	}
	clone := *t
	clone.ll = p.nativeStorageLLVMType(t.raw.Type, t.ll)
	ret := &clone
	p.nativeStorage[ret] = struct{}{}
	return ret
}

func (p Program) nativeStorageLLVMType(raw types.Type, fallback llvm.Type) llvm.Type {
	if p.GoWordSize() <= p.PointerSize() {
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
	if !p.isNativeStorage(t) {
		return t
	}
	return p.rawType(t.raw.Type)
}

func (p Program) childStorageType(parent Type, child types.Type) Type {
	t := p.rawType(child)
	if p.isNativeStorage(parent) || p.hasNativeTypeLayout(parent.raw.Type) || p.hasNativeTypeLayout(child) {
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
	if p.needsWidePointerStorage(t) {
		return p.widePointerStorageType()
	}
	return t.ll
}

func (p Program) requireStorageAlignment(value llvm.Value, t Type) {
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
	if b.Prog.needsWidePointerStorage(t) {
		storage := b.Prog.widePointerStorageType()
		parts := storage.StructElementTypes()
		stored := llvm.Undef(storage)
		stored = b.impl.CreateInsertValue(stored, value, 0, "")
		return b.impl.CreateInsertValue(stored, llvm.ConstNull(parts[1]), 1, "")
	}
	return b.fitLLVMValue(value, b.Prog.withoutNativeStorage(t), t.ll)
}

func (b Builder) fromStorageValue(t Type, value llvm.Value) llvm.Value {
	if b.Prog.needsWidePointerStorage(t) {
		return b.impl.CreateExtractValue(value, 0, "")
	}
	logical := b.Prog.withoutNativeStorage(t)
	return b.fitLLVMValue(value, t, logical.ll)
}

func (p Program) toStorageConstant(t Type, value llvm.Value) llvm.Value {
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
		return llvm.CreatePtrToInt(b.impl, value, b.Prog.tyInt())
	}
	return b.toStorageValue(t, value)
}

func (b Builder) fromAtomicStorageValue(t Type, value llvm.Value) llvm.Value {
	if b.Prog.needsWidePointerStorage(t) {
		return llvm.CreateIntToPtr(b.impl, value, t.ll)
	}
	return b.fromStorageValue(t, value)
}
