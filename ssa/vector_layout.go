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
)

type vectorSourceLayout struct{ size, align int64 }

// vectorLayout is the source-checker's counterpart of native vector storage.
// It uses Go metadata only: types may still be under construction, and LLVM
// types must not be cached before the checker completes them. Unaffected Go
// types continue to use the existing sizes implementation.
func (p *goProgram) vectorLayout(t types.Type, native bool) (vectorSourceLayout, bool) {
	prog := Program(p)
	if index, ok := simdTagIndex(t); ok {
		return vectorSourceLayout{0, p.simdAlignments[index]}, true
	}
	switch t := types.Unalias(t).(type) {
	case *types.Named:
		// Ordinary named types need neither a linkname nor native metadata.
		// Keep this check type-only: callers may query types before imports or
		// calling conventions have been registered.
		if !prog.containsVectorStorage(t) {
			return vectorSourceLayout{}, false
		}
		return p.vectorLayout(t.Underlying(), prog.hasNativeStructLayout(t))
	case *types.Array:
		// Element layout propagates SIMD alignment through arrays, including
		// zero-length arrays and defined types.
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
		if !prog.containsVectorStorage(t) {
			return vectorSourceLayout{}, false
		}
		fields := make([]*types.Var, t.NumFields())
		for i := range fields {
			fields[i] = t.Field(i)
		}
		layout, _ := p.vectorFieldsLayout(fields, native)
		return layout, true
	}
	return vectorSourceLayout{}, false
}

func (p *goProgram) vectorStructLayout(fields []*types.Var, native bool) (vectorSourceLayout, []int64, bool) {
	// Inspect types before allocating layout scratch space. SIMD layout must
	// not depend on import order or prior LLVM type construction.
	prog := Program(p)
	changed := false
	for _, field := range fields {
		if prog.containsVectorStorage(field.Type()) {
			changed = true
			break
		}
	}
	if !changed {
		return vectorSourceLayout{}, nil, false
	}
	layout, offsets := p.vectorFieldsLayout(fields, native)
	return layout, offsets, true
}

// vectorFieldsLayout computes a layout after the caller has found vector
// storage. Keep the presence check outside to avoid scratch allocations for
// ordinary fields and a repeated scan when vectorLayout already checked them.
func (p *goProgram) vectorFieldsLayout(fields []*types.Var, native bool) (vectorSourceLayout, []int64) {
	layouts := make([]vectorSourceLayout, len(fields))
	changedFields := make([]bool, len(fields))
	for i, field := range fields {
		layout, ok := p.vectorLayout(field.Type(), native)
		layouts[i], changedFields[i] = layout, ok
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
	return result, offsets
}

func vectorLayoutAlign(size, alignment int64) int64 {
	if size < 0 || size > math.MaxInt64-(alignment-1) {
		return -1
	}
	return align(size, alignment)
}
