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

var valueAttributeKinds = attributeKinds(
	"nonnull", "noalias", "noundef", "align", "dereferenceable", "dereferenceable_or_null",
	"allocptr", "allocalign", "readonly", "writeonly", "captures", "nofree", "returned", "range",
	"signext", "zeroext", "inreg", "sret", "byval",
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

// Value properties follow only an unchanged value. An aggregate's attributes
// cannot be copied to one of its register pieces or to a new storage pointer.
// The latter receives byval/sret from transformFuncType, never result noalias.
func copyValueAttributes(info *FuncInfo, paramMap []int, variadicCount, newFixedCount int, get func(int, uint) llvm.Attribute, add func(int, llvm.Attribute)) {
	copyIndex := func(oldIndex, newIndex int, kinds []uint) {
		for _, kind := range kinds {
			if kind == returnedKind && info.Return.Kind != AttrNone {
				continue
			}
			attr := get(oldIndex, kind)
			if attr.IsNil() || (info.Return.Kind == AttrPointer && !llvmattr.ValidOnSRetParameter(attr)) {
				continue
			}
			add(newIndex, attr)
		}
	}
	if info.Return.Kind == AttrNone {
		copyIndex(0, 0, valueAttributeKinds)
	}
	for i, ti := range info.Params {
		if paramMap[i] == 0 {
			continue
		}
		copyIndex(i+1, paramMap[i], closureEnvAttributeKinds)
		if ti.Kind == AttrNone {
			copyIndex(i+1, paramMap[i], valueAttributeKinds)
		}
	}
	// Variadic operands are already promoted and are appended unchanged.
	for i := 1; i <= variadicCount; i++ {
		copyIndex(len(paramMap)+i, newFixedCount+i, valueAttributeKinds)
	}
}

func copyFunctionAttributes(ctx llvm.Context, from, to llvm.Value, info *FuncInfo, paramMap []int) {
	for _, attr := range from.GetFunctionAttributes() {
		if attr = remapFunctionAttribute(ctx, attr, info, paramMap); !attr.IsNil() {
			to.AddAttributeAtIndex(-1, attr)
		}
	}
	copyValueAttributes(info, paramMap, 0, 0, from.GetEnumAttributeAtIndex, to.AddAttributeAtIndex)
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
	variadicCount := from.OperandsCount() - 1 - len(info.Params)
	copyValueAttributes(info, paramMap, variadicCount, to.CalledFunctionType().ParamTypesCount(), from.GetCallSiteEnumAttribute, to.AddCallSiteAttribute)
}
