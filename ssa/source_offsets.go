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

// SourceOffsets computes offsets with the actual declaring type. Unlike
// types.Sizes.Offsetsof, this preserves the distinction between Go and native
// owners that share their underlying struct and field objects. It is safe
// during type checking: it never materializes or caches an LLVM type.
func (p Program) SourceOffsets(owner types.Type) []int64 {
	_, offsets := (*goProgram)(p).offsetStructLayout(owner, false)
	return offsets
}

// raw records suppression of Go closure conversion by an enclosing native
// declaration. nativeSlot records only the physical width of this field.
// A nested struct owns its packing policy even when its fields are raw.
func (p *goProgram) offsetFieldLayout(t types.Type, raw, nativeSlot bool) sourceLayout {
	prog := Program(p)
	raw = raw || prog.hasLegacyNativeTypeLayout(t)
	word := int64(prog.GoWordSize())
	slot := word
	if nativeSlot {
		slot = int64(prog.PointerSize())
	}
	if index, ok := simdTagIndex(t); ok {
		return sourceLayout{0, p.simdAlignments[index]}
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		switch u.Kind() {
		case types.String:
			return sourceLayout{2 * word, word}
		case types.UnsafePointer:
			return sourceLayout{slot, int64(prog.PointerSize())}
		case types.Uintptr:
			return sourceLayout{slot, slot}
		}
		return prog.hostBasicLayouts[u.Kind()]
	case *types.Pointer, *types.Map, *types.Chan:
		return sourceLayout{slot, int64(prog.PointerSize())}
	case *types.Signature:
		if raw {
			return sourceLayout{slot, int64(prog.PointerSize())}
		}
		return sourceLayout{2 * word, word}
	case *types.Interface:
		return sourceLayout{2 * word, word}
	case *types.Slice:
		return sourceLayout{3 * word, word}
	case *types.Array:
		if _, named := types.Unalias(t).(*types.Named); named {
			nativeSlot = nativeSlot || prog.hasLegacyNativeTypeLayout(t)
		}
		return arraySourceLayout(p.offsetFieldLayout(u.Elem(), raw, nativeSlot), u.Len())
	case *types.Struct:
		layout, _ := p.offsetStructLayout(t, raw)
		return layout
	}
	return sourceLayout{p.Sizeof(t), p.Alignof(t)}
}

func (p *goProgram) offsetStructLayout(owner types.Type, raw bool) (sourceLayout, []int64) {
	prog := Program(p)
	st := owner.Underlying().(*types.Struct)
	native := prog.hasLegacyNativeTypeLayout(owner)
	raw = raw || native || IsClosure(st)
	host := hasHostLayout(owner)
	reconcile := !native && prog.reconcilesStructLayout(st)
	return aggregateSourceLayout(st.NumFields(), func(i int) sourceLayout {
		t := st.Field(i).Type()
		l := p.offsetFieldLayout(t, raw, native)
		if reconcile {
			l.align = p.structFieldAlignment(t, l.align, host)
		}
		return l
	}, reconcile)
}
