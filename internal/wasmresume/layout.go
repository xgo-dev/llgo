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

import "github.com/xgo-dev/llvm"

const frameHeaderFields = 3

type frameLayout struct {
	plan         framePlan
	typ          llvm.Type
	size         uint64
	alignment    int
	fields       []int
	unwindOffset uint64
}

func layoutFrames(mod llvm.Module, targetData llvm.TargetData) ([]frameLayout, error) {
	plans, err := planFrames(mod)
	if err != nil {
		return nil, err
	}
	ctx := mod.Context()
	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	header := []llvm.Type{ptr, ptr, ctx.Int32Type()}

	layouts := make([]frameLayout, len(plans))
	for i, plan := range plans {
		fields := append([]llvm.Type(nil), header...)
		fieldIndices := make([]int, len(plan.slots)+1)
		headerType := ctx.StructType(header, false)
		frameAlign := targetData.ABITypeAlignment(headerType)
		for _, slot := range plan.slots {
			align := targetData.ABITypeAlignment(slot.typ)
			if slot.kind == slotAlloca && slot.value.Alignment() > align {
				align = slot.value.Alignment()
			}
			if align > frameAlign {
				frameAlign = align
			}
			withSlot := append(append([]llvm.Type(nil), fields...), slot.typ)
			naturalOffset := targetData.ElementOffset(
				ctx.StructType(withSlot, false), len(withSlot)-1,
			)
			if padding := alignmentPadding(naturalOffset, uint64(align)); padding != 0 {
				fields = append(fields, llvm.ArrayType(ctx.Int8Type(), int(padding)))
			}
			fieldIndices[slot.id] = len(fields)
			fields = append(fields, slot.typ)
		}
		typ := ctx.StructType(fields, false)
		var unwindOffset uint64
		if plan.unwindSlot != 0 {
			unwindOffset = targetData.ElementOffset(typ, fieldIndices[plan.unwindSlot])
		}
		layouts[i] = frameLayout{
			plan:         plan,
			typ:          typ,
			size:         targetData.TypeAllocSize(typ),
			alignment:    frameAlign,
			fields:       fieldIndices,
			unwindOffset: unwindOffset,
		}
	}
	return layouts, nil
}

func (l frameLayout) fieldIndex(slotID uint32) int {
	if slotID == 0 || int(slotID) >= len(l.fields) {
		return -1
	}
	return l.fields[slotID]
}

func alignmentPadding(offset, align uint64) uint64 {
	return -offset & (align - 1)
}
