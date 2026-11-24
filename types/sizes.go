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

package types

import (
	gotypes "go/types"

	"github.com/xgo-dev/llgo/target"
)

// Sizes computes target-accurate byte sizes, alignments, struct field offsets,
// and GC pointer offset maps for LLGo Types in pure Go without LLVM TargetData.
type Sizes struct {
	Spec target.Spec
}

// NewSizes returns a Sizes calculator for the resolved target specification.
func NewSizes(spec target.Spec) *Sizes {
	if spec.WordSize == 0 || spec.PtrSize == 0 {
		spec = target.ResolveSpec(spec)
	}
	return &Sizes{Spec: spec}
}

func alignUp(offset, align int64) int64 {
	if align <= 1 {
		return offset
	}
	return (offset + align - 1) &^ (align - 1)
}

func (s *Sizes) basicLayout(kind gotypes.BasicKind, host bool, native bool) (size, align int64) {
	word := int64(s.Spec.WordSize)
	ptr := int64(s.Spec.PtrSize)
	maxAlign := int64(s.Spec.MaxAlign)
	align64 := maxAlign
	if host || s.Spec.HostUsesNativeGo {
		align64 = int64(s.Spec.HostAlign64)
	}
	switch kind {
	case gotypes.Bool, gotypes.Int8, gotypes.Uint8:
		return 1, 1
	case gotypes.Int16, gotypes.Uint16:
		return 2, 2
	case gotypes.Int32, gotypes.Uint32, gotypes.Float32:
		return 4, 4
	case gotypes.Int64, gotypes.Uint64, gotypes.Float64:
		return 8, align64
	case gotypes.Complex64:
		return 8, 4
	case gotypes.Complex128:
		return 16, align64
	case gotypes.Int, gotypes.Uint:
		return word, word
	case gotypes.Uintptr, gotypes.UnsafePointer:
		if native && s.Spec.UsesWideGoStorage() {
			return ptr, ptr
		}
		return word, word
	default:
		return 0, 1
	}
}

func (s *Sizes) layout(t Type, host bool) (size, align int64) {
	word := int64(s.Spec.WordSize)
	ptr := int64(s.Spec.PtrSize)
	switch u := t.Underlying().(type) {
	case *Basic:
		return s.basicLayout(u.Kind, host, u.PtrKind == NativePtr)
	case *Pointer:
		if u.Kind == NativePtr && s.Spec.UsesWideGoStorage() {
			return ptr, ptr
		}
		return word, word
	case *Func:
		if u.CallConv.IsNative() && s.Spec.UsesWideGoStorage() {
			return ptr, ptr
		}
		return word, word
	case *Closure:
		// First-class Go closure: {codePtr, envPtr} = 2 words.
		return 2 * word, word
	case *String:
		return 2 * word, word
	case *Interface:
		return 2 * word, word
	case *Slice:
		return 3 * word, word
	case *Map, *Chan:
		return word, word
	case *SIMDVec:
		es, _ := s.basicLayout(u.Elem.Kind, true, true)
		total := es * int64(u.Len)
		return total, total
	case *Array:
		if u.Len == 0 {
			_, ea := s.layout(u.Elem, host)
			return 0, ea
		}
		es, ea := s.layout(u.Elem, host)
		return alignUp(es, ea) * u.Len, ea
	case *Struct:
		stSize, stAlign, _ := s.structLayout(u)
		return stSize, stAlign
	case *Tuple:
		var off int64
		maxA := int64(1)
		for _, e := range u.Elems {
			es, ea := s.layout(e.Type, false)
			if ea > maxA {
				maxA = ea
			}
			off = alignUp(off, ea) + es
		}
		return alignUp(off, maxA), maxA
	default:
		return 0, 1
	}
}

func (s *Sizes) structLayout(st *Struct) (size, align int64, offsets []int64) {
	offsets = make([]int64, len(st.Fields))
	var off int64
	maxA := int64(1)
	padTrailingZero := !st.HostLayout && !st.NativeStorage && !s.Spec.HostUsesNativeGo
	for i, f := range st.Fields {
		fs, fa := s.layout(f.Type, st.HostLayout)
		if fa > maxA {
			maxA = fa
		}
		off = alignUp(off, fa)
		offsets[i] = off
		off += fs
	}
	// Go pads a non-empty struct ending in a zero-sized field by 1 byte so a
	// pointer to the final field does not point past the allocation.
	if padTrailingZero && off > 0 && len(st.Fields) > 0 {
		lastSize, _ := s.layout(st.Fields[len(st.Fields)-1].Type, st.HostLayout)
		if lastSize == 0 {
			off++
		}
	}
	return alignUp(off, maxA), maxA, offsets
}

// Sizeof returns the byte size of t.
func (s *Sizes) Sizeof(t Type) int64 {
	sz, _ := s.layout(t, false)
	return sz
}

// Alignof returns the byte alignment of t (always >= 1).
func (s *Sizes) Alignof(t Type) int64 {
	_, al := s.layout(t, false)
	return al
}

// Offsetsof returns the byte offset of each field in st.
func (s *Sizes) Offsetsof(st *Struct) []int64 {
	_, _, offsets := s.structLayout(st)
	return offsets
}

// PtrOffsets returns the sorted byte offsets of every GC-traced pointer word
// inside t. NativePtr fields are excluded, enabling precise stack maps and
// object bitmaps for custom GC implementations.
func (s *Sizes) PtrOffsets(t Type) []int64 {
	var out []int64
	s.appendPtrOffsets(t, 0, &out)
	return out
}

// PtrDataSize returns the prefix size in bytes containing all GC-traced
// pointers in t (matching Go's runtime/abi.Type.PtrBytes contract).
func (s *Sizes) PtrDataSize(t Type) int64 {
	offsets := s.PtrOffsets(t)
	if len(offsets) == 0 {
		return 0
	}
	return offsets[len(offsets)-1] + int64(s.Spec.WordSize)
}

func (s *Sizes) appendPtrOffsets(t Type, base int64, out *[]int64) {
	if !t.HasPtrData() {
		return
	}
	word := int64(s.Spec.WordSize)
	switch u := t.Underlying().(type) {
	case *Basic:
		if u.Kind == gotypes.UnsafePointer && u.PtrKind == GoPtr {
			*out = append(*out, base)
		}
	case *Pointer:
		if u.Kind == GoPtr {
			*out = append(*out, base)
		}
	case *Func:
		if u.CallConv == GoDecl {
			*out = append(*out, base)
		}
	case *Closure:
		// {codePtr (not heap-allocated), envPtr (GC-traced heap pointer)}
		*out = append(*out, base+word)
	case *String, *Slice:
		*out = append(*out, base)
	case *Interface:
		*out = append(*out, base, base+word)
	case *Map, *Chan:
		*out = append(*out, base)
	case *Array:
		es, ea := s.layout(u.Elem, false)
		stride := alignUp(es, ea)
		for i := int64(0); i < u.Len; i++ {
			s.appendPtrOffsets(u.Elem, base+i*stride, out)
		}
	case *Struct:
		_, _, offs := s.structLayout(u)
		for i, f := range u.Fields {
			s.appendPtrOffsets(f.Type, base+offs[i], out)
		}
	}
}
