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

// containsNativeAggregate follows storage edges only. A pointer to a native
// record does not change the layout of the containing Go object.
func (p Program) containsNativeAggregate(t types.Type) bool {
	if !p.packageSyntax.hasNativeTypeMetadata.Load() {
		return false
	}
	if p.hasLegacyNativeTypeLayout(t) {
		return true
	}
	switch t := t.Underlying().(type) {
	case *types.Array:
		return p.containsNativeAggregate(t.Elem())
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			if p.containsNativeAggregate(t.Field(i).Type()) {
				return true
			}
		}
	}
	return false
}

// representationLayout runs during checking, before LLVM named types may be
// materialized. Recompute enclosing aggregates from their field layouts: adding
// a closure-size correction to gc's offsets cannot account for native fields.
func (p *goProgram) representationLayout(t types.Type) (sourceLayout, bool) {
	prog := Program(p)
	if !prog.containsNativeAggregate(t) {
		return sourceLayout{}, false
	}
	switch u := t.Underlying().(type) {
	case *types.Struct:
		layout, _ := p.representationStructLayout(u, prog.hasLegacyNativeTypeLayout(t))
		return layout, true
	case *types.Array:
		return arraySourceLayout(p.representationFieldLayout(u.Elem(), prog.RepresentationOf(t).Storage == NativeRepresentation), u.Len()), true
	}
	return sourceLayout{}, false
}

func (p *goProgram) representationStructLayout(st *types.Struct, native bool) (sourceLayout, []int64) {
	return aggregateSourceLayout(st.NumFields(), func(i int) sourceLayout {
		return p.representationFieldLayout(st.Field(i).Type(), native)
	}, !native)
}

func (p *goProgram) representationFieldLayout(t types.Type, native bool) sourceLayout {
	prog := Program(p)
	if native {
		switch u := t.Underlying().(type) {
		case *types.Signature, *types.Pointer, *types.Chan, *types.Map:
			ptr := int64(prog.PointerSize())
			return sourceLayout{ptr, ptr}
		case *types.Basic:
			if u.Kind() == types.Uintptr || u.Kind() == types.UnsafePointer {
				ptr := int64(prog.PointerSize())
				return sourceLayout{ptr, ptr}
			}
			if u.Kind() == types.Int || u.Kind() == types.Uint {
				word := int64(prog.GoWordSize())
				return sourceLayout{word, word}
			}
			if u.Kind() != types.String && u.Kind() > types.Invalid && u.Kind() <= types.UnsafePointer {
				return prog.hostBasicLayouts[u.Kind()]
			}
		case *types.Array:
			return arraySourceLayout(p.representationFieldLayout(u.Elem(), true), u.Len())
		}
	}
	return sourceLayout{p.Sizeof(t), p.Alignof(t)}
}
