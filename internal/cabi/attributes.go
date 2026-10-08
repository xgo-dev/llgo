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
var callFunctionAttributeNames = []string{
	"noreturn", "nounwind", "willreturn", "nofree", "nosync", "nocallback",
	"memory", "allocsize", "returns_twice", "convergent", "noduplicate",
}
var valueAttributeNames = []string{
	"nonnull", "noalias", "noundef", "align", "dereferenceable", "dereferenceable_or_null",
	"readonly", "writeonly", "captures", "nofree", "returned", "range",
	"signext", "zeroext", "inreg", "nest", "swiftself", "sret", "byval",
}

// remapFunctionAttribute adjusts contracts whose meaning depends on the ABI.
// A by-reference input adds a read; sret adds a write. In particular, copying
// memory(none/read) unchanged to a function returning via sret is unsound.
func remapFunctionAttribute(ctx llvm.Context, attr llvm.Attribute, info *FuncInfo, paramMap []int) llvm.Attribute {
	if !attr.IsEnum() {
		return attr
	}
	switch uint(attr.GetEnumKind()) {
	case llvm.AttributeKindID("memory"):
		effects := attr.GetEnumValue()
		for _, ti := range info.Params {
			if ti.Kind == AttrPointer {
				effects |= llvmattr.MemoryArgRead
			}
		}
		if info.Return.Kind == AttrPointer {
			effects |= llvmattr.MemoryArgWrite
		}
		return ctx.CreateEnumAttribute(llvm.AttributeKindID("memory"), effects)
	case llvm.AttributeKindID("allocsize"):
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
		return ctx.CreateEnumAttribute(llvm.AttributeKindID("allocsize"), llvmattr.AllocSize(first, second))
	}
	return attr
}

// Value properties follow only an unchanged value. An aggregate's attributes
// cannot be copied to one of its register pieces or to a new storage pointer.
// The latter receives byval/sret from transformFuncType, never result noalias.
func copyValueAttributes(info *FuncInfo, paramMap []int, get func(int, uint) llvm.Attribute, add func(int, llvm.Attribute)) {
	copyIndex := func(oldIndex, newIndex int) {
		for _, name := range valueAttributeNames {
			if name == "returned" && info.Return.Kind != AttrNone {
				continue
			}
			if attr := get(oldIndex, llvm.AttributeKindID(name)); !attr.IsNil() {
				add(newIndex, attr)
			}
		}
	}
	if info.Return.Kind == AttrNone {
		copyIndex(0, 0)
	}
	for i, ti := range info.Params {
		if ti.Kind == AttrNone && paramMap[i] != 0 {
			copyIndex(i+1, paramMap[i])
		}
	}
}

func copyFunctionAttributes(ctx llvm.Context, from, to llvm.Value, info *FuncInfo, paramMap []int) {
	for _, attr := range from.GetFunctionAttributes() {
		if attr = remapFunctionAttribute(ctx, attr, info, paramMap); !attr.IsNil() {
			to.AddAttributeAtIndex(-1, attr)
		}
	}
	copyValueAttributes(info, paramMap, from.GetEnumAttributeAtIndex, to.AddAttributeAtIndex)
}

func copyCallAttributes(ctx llvm.Context, from, to llvm.Value, info *FuncInfo, paramMap []int) {
	for _, name := range callFunctionAttributeNames {
		attr := from.GetCallSiteEnumAttribute(-1, llvm.AttributeKindID(name))
		if attr.IsNil() {
			continue
		}
		if attr = remapFunctionAttribute(ctx, attr, info, paramMap); !attr.IsNil() {
			to.AddCallSiteAttribute(-1, attr)
		}
	}
	copyValueAttributes(info, paramMap, from.GetCallSiteEnumAttribute, to.AddCallSiteAttribute)
}
