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

package cabi

import (
	"slices"

	"github.com/xgo-dev/llgo/internal/llvmattr"
	"github.com/xgo-dev/llvm"
)

// These semantic attributes are supported on both functions and call sites.
// The binding exposes enumeration for function attributes but only individual
// lookups for call attributes. Keep the call list here, independent of runtime
// symbol names, so indirect calls and non-runtime functions use the same rules.
var callFunctionAttributeKinds = attributeKinds(
	"cold", "minsize", "allockind", "noreturn", "nounwind", "willreturn", "nofree", "nosync", "nocallback",
	"speculatable", "memory", "allocsize", "returns_twice", "convergent", "noduplicate",
)
var closureEnvAttributeKinds = attributeKinds("nest", "swiftself")

// The binding can only probe call-site attributes by kind. Restrict probes
// for scalar integers and pointers to attributes LLVM accepts on those types.
// Other types use the full list (e.g. align or range on vector elements).
var (
	commonValueKinds = attributeKinds("noundef", "nofree", "writeonly", "returned", "inreg", "swiftself")
	intOnlyKinds     = attributeKinds("allocalign", "signext", "zeroext", "range")
	ptrValueKinds    = slices.Concat(commonValueKinds, attributeKinds(
		"nonnull", "noalias", "align", "dereferenceable", "dereferenceable_or_null",
		"allocptr", "readonly", "captures", "sret", "byval", "nest",
	))
	intValueKinds = slices.Concat(commonValueKinds, intOnlyKinds)
	allValueKinds = slices.Concat(ptrValueKinds, intOnlyKinds)
)

func attributeKinds(names ...string) []uint {
	kinds := make([]uint, len(names))
	for i, name := range names {
		kinds[i] = llvm.AttributeKindID(name)
	}
	return kinds
}

var (
	memoryKind       = llvm.AttributeKindID("memory")
	allocSizeKind    = llvm.AttributeKindID("allocsize")
	returnedKind     = llvm.AttributeKindID("returned")
	speculatableKind = llvm.AttributeKindID("speculatable")
)

// remapFunctionAttribute adjusts contracts whose meaning depends on the ABI.
// Indirect parameters add memory accesses, and sret writes the result.
func remapFunctionAttribute(ctx llvm.Context, attr llvm.Attribute, info *FuncInfo, paramMap []int) llvm.Attribute {
	if !attr.IsEnum() {
		return attr
	}
	switch uint(attr.GetEnumKind()) {
	case speculatableKind:
		if info.Return.Kind == AttrPointer {
			return llvm.Attribute{}
		}
		for _, ti := range info.Params {
			if ti.Kind == AttrPointer {
				return llvm.Attribute{}
			}
		}
	case memoryKind:
		effects := attr.GetEnumValue()
		for _, ti := range info.Params {
			if ti.Kind == AttrPointer {
				effects |= llvmattr.MemoryArgRead
				if !ti.hasNativeLayoutConversion() {
					// The callee can reuse its argument storage as a writable
					// parameter home, including in a separately compiled module.
					effects |= llvmattr.MemoryArgWrite
				}
			}
		}
		if info.Return.Kind == AttrPointer {
			effects |= llvmattr.MemoryArgWrite
		}
		return ctx.CreateEnumAttribute(memoryKind, effects)
	case allocSizeKind:
		if info.Return.Kind == AttrPointer {
			return llvm.Attribute{} // The rewritten function returns void.
		}
		// allocsize uses zero-based indices, unlike LLVM attribute indices.
		element, count := llvmattr.AllocSizeArgs(attr.GetEnumValue())
		remap := func(index uint32) (uint32, bool) {
			if uint64(index) >= uint64(len(paramMap)) || paramMap[index] == 0 || info.Params[index].Kind != AttrNone {
				return 0, false
			}
			return uint32(paramMap[index] - 1), true
		}
		first, ok := remap(element)
		if !ok {
			return llvm.Attribute{}
		}
		second := count
		if second != llvmattr.AllocSizeNoCount {
			second, ok = remap(second)
			if !ok {
				return llvm.Attribute{}
			}
		}
		return ctx.CreateEnumAttribute(allocSizeKind, llvmattr.AllocSize(first, second))
	}
	return attr
}

// Register pieces and indirect storage receive only the closure ABI markers.
// Semantic properties follow a value only when it survives lowering intact.
func visitValueIndices(info *FuncInfo, paramMap []int, visit func(old, new int, ti *TypeInfo)) {
	if info.Return.Kind == AttrNone {
		visit(0, 0, info.Return)
	}
	for i, index := range paramMap {
		if index != 0 {
			visit(i+1, index, info.Params[i])
		}
	}
}

func keepValueAttribute(info *FuncInfo, kind uint, attr llvm.Attribute) bool {
	if kind == returnedKind && info.Return.Kind != AttrNone {
		return false
	}
	return info.Return.Kind != AttrPointer || llvmattr.ValidOnSRetParameter(attr)
}

func copyFunctionAttributes(ctx llvm.Context, from, to llvm.Value, info *FuncInfo, paramMap []int) {
	for _, attr := range from.GetFunctionAttributes() {
		if attr = remapFunctionAttribute(ctx, attr, info, paramMap); !attr.IsNil() {
			to.AddAttributeAtIndex(-1, attr)
		}
	}
	visitValueIndices(info, paramMap, func(old, new int, ti *TypeInfo) {
		// Enumeration takes one query for an empty index, regardless of type.
		kinds := allValueKinds
		if ti.Kind != AttrNone {
			kinds = closureEnvAttributeKinds
		}
		for _, attr := range from.GetAttributesAtIndex(old) {
			if attr.IsString() {
				continue
			}
			kind := uint(attr.GetEnumKind())
			if slices.Contains(kinds, kind) && keepValueAttribute(info, kind, attr) {
				to.AddAttributeAtIndex(new, attr)
			}
		}
	})
}

func copyCallAttributes(ctx llvm.Context, from, to llvm.Value, info *FuncInfo, paramMap []int) {
	for _, kind := range callFunctionAttributeKinds {
		attr := from.GetCallSiteEnumAttribute(-1, kind)
		if attr.IsNil() {
			continue
		}
		if attr = remapFunctionAttribute(ctx, attr, info, paramMap); !attr.IsNil() {
			to.AddCallSiteAttribute(-1, attr)
		}
	}
	if family := from.GetCallSiteStringAttribute(-1, "alloc-family"); !family.IsNil() {
		to.AddCallSiteAttribute(-1, family)
	}
	copyIndex := func(old, new int, kinds []uint) {
		for _, kind := range kinds {
			attr := from.GetCallSiteEnumAttribute(old, kind)
			if !attr.IsNil() && keepValueAttribute(info, kind, attr) {
				to.AddCallSiteAttribute(new, attr)
			}
		}
	}
	visitValueIndices(info, paramMap, func(old, new int, ti *TypeInfo) {
		if ti.Kind != AttrNone {
			copyIndex(old, new, closureEnvAttributeKinds)
			return
		}
		switch ti.Type.TypeKind() {
		case llvm.IntegerTypeKind:
			copyIndex(old, new, intValueKinds)
		case llvm.PointerTypeKind:
			copyIndex(old, new, ptrValueKinds)
		default:
			copyIndex(old, new, allValueKinds)
		}
	})
	// The call operands after the fixed parameters are already promoted.
	if info.Type.IsFunctionVarArg() {
		fixed := len(paramMap)
		shift := to.CalledFunctionType().ParamTypesCount() - fixed
		for i, end := fixed+1, from.OperandsCount(); i < end; i++ {
			copyIndex(i, i+shift, allValueKinds)
		}
	}
}
