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

import "go/types"

// alignedLayout is the source-checker's counterpart of explicitly aligned and vector storage.
// It uses Go metadata only: types may still be under construction, and LLVM
// types must not be cached before the checker completes them. Unaffected Go
// types continue to use the existing sizes implementation.
func (p *goProgram) alignedLayout(t types.Type, native bool) (sourceLayout, bool) {
	prog := Program(p)
	if a := explicitAlignment(t); a != 0 {
		return sourceLayout{0, a}, true
	}
	if index, ok := simdTagIndex(t); ok {
		return sourceLayout{0, p.simdAlignments[index]}, true
	}
	switch t := types.Unalias(t).(type) {
	case *types.Named:
		// Ordinary named types need neither a linkname nor native metadata.
		// Keep this check type-only: callers may query types before imports or
		// calling conventions have been registered.
		if !prog.containsAlignedStorage(t) {
			return sourceLayout{}, false
		}
		return p.alignedLayout(t.Underlying(), prog.hasNativeStructLayout(t))
	case *types.Array:
		// Element layout propagates alignment through arrays, including
		// zero-length arrays and defined types.
		elem, changed := p.alignedLayout(t.Elem(), false)
		if !changed {
			return sourceLayout{}, false
		}
		return arraySourceLayout(elem, t.Len()), true
	case *types.Struct:
		if !prog.containsAlignedStorage(t) {
			return sourceLayout{}, false
		}
		fields := make([]*types.Var, t.NumFields())
		for i := range fields {
			fields[i] = t.Field(i)
		}
		layout, _ := p.alignedFieldsLayout(fields, native)
		return layout, true
	}
	return sourceLayout{}, false
}

func (p *goProgram) alignedStructLayout(fields []*types.Var, native bool) (sourceLayout, []int64, bool) {
	// Inspect types before allocating layout scratch space. Aligned layout must
	// not depend on import order or prior LLVM type construction.
	prog := Program(p)
	changed := false
	for _, field := range fields {
		if prog.containsAlignedStorage(field.Type()) {
			changed = true
			break
		}
	}
	if !changed {
		return sourceLayout{}, nil, false
	}
	layout, offsets := p.alignedFieldsLayout(fields, native)
	return layout, offsets, true
}

// alignedFieldsLayout computes a layout after the caller has found aligned
// storage. Keep the presence check outside to avoid scratch allocations for
// ordinary fields and a repeated scan when alignedLayout already checked them.
func (p *goProgram) alignedFieldsLayout(fields []*types.Var, native bool) (sourceLayout, []int64) {
	explicit := false
	for _, field := range fields {
		explicit = explicit || requiredExplicitAlignment(field.Type()) != 0
	}
	layouts := make([]sourceLayout, len(fields))
	changedFields := make([]bool, len(fields))
	for i, field := range fields {
		layout, ok := p.alignedLayout(field.Type(), native)
		layouts[i], changedFields[i] = layout, ok
	}
	for i, field := range fields {
		if !changedFields[i] {
			layout := p.unmarkedAlignedFieldLayout(field.Type(), native, explicit)
			if native {
				// A native function field is a code pointer, not a Go closure.
				if _, ok := field.Type().Underlying().(*types.Signature); ok {
					layout.size, layout.align = int64(p.ptrSize), int64(p.ptrSize)
				}
			}
			layouts[i] = layout
		}
	}
	return aggregateSourceLayout(len(layouts), func(i int) sourceLayout { return layouts[i] }, !native)
}

// Inside an explicitly aligned aggregate, preserve the existing physical
// layout of nested ordinary records on targets such as ARM32. Those records
// can still differ from the baseline Go source sizes; do not shift a following
// field into their storage. Direct scalar fields retain their Go alignment.
func (p *goProgram) unmarkedAlignedFieldLayout(t types.Type, native, explicit bool) sourceLayout {
	if explicit {
		if native {
			return p.hostFieldLayout(t, true)
		}
		if Program(p).hostUsesNativeGoLayout() {
			switch t.Underlying().(type) {
			case *types.Struct, *types.Array:
				return p.hostNativeLayout(t, false)
			}
		}
	}
	return sourceLayout{p.Sizeof(t), p.Alignof(t)}
}
