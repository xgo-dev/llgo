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
	"math"
	"unsafe"
)

type vectorSourceLayout struct{ size, align int64 }

// vectorLayout is the source-checker's counterpart of native vector storage.
// It uses Go metadata only: types may still be under construction, and LLVM
// types must not be cached before the checker completes them. Unaffected Go
// types continue to use the existing sizes implementation.
func (p *goProgram) vectorLayout(t types.Type, native bool) (vectorSourceLayout, bool) {
	prog := Program(unsafe.Pointer(p))
	if _, ok := prog.cVectorArray(t); ok {
		return vectorSourceLayout{16, 16}, true
	}
	if native {
		if _, ok := SIMDVectorShape(t); ok {
			return vectorSourceLayout{16, 16}, true
		}
	}
	switch t := types.Unalias(t).(type) {
	case *types.Named:
		return p.vectorLayout(t.Underlying(), prog.hasNativeStructLayout(t))
	case *types.Array:
		// Array elements use their own storage representation, including the
		// ordinary Go storage of archsimd, even in a native enclosing struct.
		elem, changed := p.vectorLayout(t.Elem(), false)
		if !changed {
			return vectorSourceLayout{}, false
		}
		size := int64(0)
		if t.Len() > 0 {
			size = -1
			if elem.size >= 0 && (elem.size == 0 || t.Len() <= math.MaxInt64/elem.size) {
				size = t.Len() * elem.size
			}
		}
		return vectorSourceLayout{size, elem.align}, true
	case *types.Struct:
		fields := make([]*types.Var, t.NumFields())
		for i := range fields {
			fields[i] = t.Field(i)
		}
		layout, _, changed := p.vectorStructLayout(fields, native)
		return layout, changed
	}
	return vectorSourceLayout{}, false
}

func (p *goProgram) vectorStructLayout(fields []*types.Var, native bool) (vectorSourceLayout, []int64, bool) {
	layouts := make([]vectorSourceLayout, len(fields))
	changedFields := make([]bool, len(fields))
	changed := false
	for i, field := range fields {
		layout, ok := p.vectorLayout(field.Type(), native)
		changed = changed || ok
		layouts[i], changedFields[i] = layout, ok
	}
	if !changed {
		return vectorSourceLayout{}, nil, false
	}
	for i, field := range fields {
		if !changedFields[i] {
			layout := vectorSourceLayout{p.Sizeof(field.Type()), p.Alignof(field.Type())}
			if native {
				// A native function field is a code pointer, not a Go closure.
				if _, ok := field.Type().Underlying().(*types.Signature); ok {
					layout.size, layout.align = int64(p.ptrSize), int64(p.ptrSize)
				}
			}
			layouts[i] = layout
		}
	}
	result := vectorSourceLayout{align: 1}
	offsets := make([]int64, len(fields))
	for i, field := range layouts {
		result.align = max(result.align, field.align)
		result.size = vectorLayoutAlign(result.size, field.align)
		offsets[i] = result.size
		if result.size < 0 || field.size < 0 || field.size > math.MaxInt64-result.size {
			result.size = -1
		} else {
			result.size += field.size
			if !native && i == len(fields)-1 && field.size == 0 && result.size > 0 {
				if result.size == math.MaxInt64 {
					result.size = -1
				} else {
					result.size++ // Go's final zero-sized field has an in-object address.
				}
			}
		}
	}
	result.size = vectorLayoutAlign(result.size, result.align)
	return result, offsets, true
}

func vectorLayoutAlign(size, alignment int64) int64 {
	if size < 0 || size > math.MaxInt64-(alignment-1) {
		return -1
	}
	return align(size, alignment)
}
