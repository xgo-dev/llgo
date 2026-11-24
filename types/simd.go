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
	"fmt"
	gotypes "go/types"
	"strings"

	"github.com/xgo-dev/llgo/target"
)

// SIMDTagIndex recognizes simd/archsimd zero-sized vector alignment markers
// (v128 -> 0, v256 -> 1, v512 -> 2).
func SIMDTagIndex(t gotypes.Type) (int, bool) {
	if t == nil {
		return 0, false
	}
	n, ok := unaliasType(t).(*gotypes.Named)
	if !ok || n.Obj().Pkg() == nil || n.Obj().Pkg().Path() != "simd/archsimd" {
		return 0, false
	}
	index := -1
	switch n.Obj().Name() {
	case "v128":
		index = 0
	case "v256":
		index = 1
	case "v512":
		index = 2
	}
	if index < 0 {
		return 0, false
	}
	st, ok := n.Underlying().(*gotypes.Struct)
	if !ok || st.NumFields() != 1 {
		return 0, false
	}
	a, ok := st.Field(0).Type().(*gotypes.Array)
	return index, ok && a.Len() == 0
}

// ContainsVectorStorage reports whether t is or transitively contains a
// simd/archsimd vector alignment marker through named types, arrays, or structs.
func ContainsVectorStorage(t gotypes.Type) bool {
	if t == nil {
		return false
	}
	if _, ok := SIMDTagIndex(t); ok {
		return true
	}
	switch u := unaliasType(t).(type) {
	case *gotypes.Named:
		return ContainsVectorStorage(u.Underlying())
	case *gotypes.Array:
		return ContainsVectorStorage(u.Elem())
	case *gotypes.Struct:
		for i := 0; i < u.NumFields(); i++ {
			if ContainsVectorStorage(u.Field(i).Type()) {
				return true
			}
		}
	}
	return false
}

// SIMDVectorShape validates numeric and mask SIMD128/256/512 storage in
// simd/archsimd and returns its underlying lane array.
func SIMDVectorShape(typ gotypes.Type) (*gotypes.Array, bool) {
	if typ == nil {
		return nil, false
	}
	named, ok := unaliasType(typ).(*gotypes.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != "simd/archsimd" {
		return nil, false
	}
	st, ok := named.Underlying().(*gotypes.Struct)
	if !ok || st.NumFields() != 2 {
		return nil, false
	}
	lanes, ok := st.Field(1).Type().(*gotypes.Array)
	if !ok {
		return nil, false
	}
	elem, ok := lanes.Elem().(*gotypes.Basic)
	if !ok {
		return nil, false
	}
	var bits int64
	switch elem.Kind() {
	case gotypes.Int8, gotypes.Uint8:
		bits = 8
	case gotypes.Int16, gotypes.Uint16:
		bits = 16
	case gotypes.Int32, gotypes.Uint32, gotypes.Float32:
		bits = 32
	case gotypes.Int64, gotypes.Uint64, gotypes.Float64:
		bits = 64
	default:
		return nil, false
	}
	name := elem.Name()
	expected := fmt.Sprintf("%s%sx%d", strings.ToUpper(name[:1]), name[1:], lanes.Len())
	if strings.HasPrefix(named.Obj().Name(), "Mask") && elem.Info()&gotypes.IsInteger != 0 && elem.Info()&gotypes.IsUnsigned == 0 {
		expected = fmt.Sprintf("Mask%dx%d", bits, lanes.Len())
	}
	width := lanes.Len() * bits
	if named.Obj().Name() != expected || (width != 128 && width != 256 && width != 512) {
		return nil, false
	}
	tag, ok := st.Field(0).Type().(*gotypes.Named)
	if !ok || tag.Obj().Pkg() != named.Obj().Pkg() || tag.Obj().Name() != fmt.Sprintf("v%d", width) {
		return nil, false
	}
	if (&gotypes.StdSizes{WordSize: 8, MaxAlign: 8}).Sizeof(tag) != 0 {
		return nil, false
	}
	return lanes, true
}

// SIMDNumericShape validates a non-mask numeric simd/archsimd vector type.
func SIMDNumericShape(typ gotypes.Type) (*gotypes.Array, bool) {
	lanes, ok := SIMDVectorShape(typ)
	if !ok {
		return nil, false
	}
	named := unaliasType(typ).(*gotypes.Named)
	return lanes, !strings.HasPrefix(named.Obj().Name(), "Mask")
}

// SIMDMaskShape validates a fixed-width simd/archsimd mask vector type.
func SIMDMaskShape(typ gotypes.Type) (*gotypes.Array, bool) {
	lanes, ok := SIMDVectorShape(typ)
	if !ok {
		return nil, false
	}
	named := unaliasType(typ).(*gotypes.Named)
	return lanes, strings.HasPrefix(named.Obj().Name(), "Mask")
}

// DefaultSIMDAlignments returns pure-Go default ABI alignments (in bytes) for
// v128, v256, and v512 vector storage on spec. Clang caps fixed-length AArch64
// C vector alignment at 16 bytes.
func DefaultSIMDAlignments(spec target.Spec) [3]int64 {
	if spec.WordSize == 0 {
		spec = target.ResolveSpec(spec)
	}
	if spec.GOARCH == "arm64" {
		return [3]int64{16, 16, 16}
	}
	return [3]int64{16, 32, 64}
}

// DefaultHostBasicLayouts computes target-accurate scalar sizes and host C ABI
// alignments for basic Go types from spec without requiring LLVM TargetData.
func DefaultHostBasicLayouts(spec target.Spec) [gotypes.UnsafePointer + 1]LayoutPair {
	if spec.WordSize == 0 || spec.PtrSize == 0 {
		spec = target.ResolveSpec(spec)
	}
	word := int64(spec.WordSize)
	ptr := int64(spec.PtrSize)
	align64 := int64(spec.HostAlign64)
	if align64 == 0 {
		align64 = int64(spec.MaxAlign)
	}
	if align64 == 0 {
		align64 = word
	}
	var out [gotypes.UnsafePointer + 1]LayoutPair
	out[gotypes.Bool] = LayoutPair{Size: 1, Align: 1}
	out[gotypes.Int8] = LayoutPair{Size: 1, Align: 1}
	out[gotypes.Uint8] = LayoutPair{Size: 1, Align: 1}
	out[gotypes.Int16] = LayoutPair{Size: 2, Align: 2}
	out[gotypes.Uint16] = LayoutPair{Size: 2, Align: 2}
	out[gotypes.Int32] = LayoutPair{Size: 4, Align: 4}
	out[gotypes.Uint32] = LayoutPair{Size: 4, Align: 4}
	out[gotypes.Float32] = LayoutPair{Size: 4, Align: 4}
	out[gotypes.Int64] = LayoutPair{Size: 8, Align: align64}
	out[gotypes.Uint64] = LayoutPair{Size: 8, Align: align64}
	out[gotypes.Float64] = LayoutPair{Size: 8, Align: align64}
	out[gotypes.Complex64] = LayoutPair{Size: 8, Align: 4}
	out[gotypes.Complex128] = LayoutPair{Size: 16, Align: align64}
	out[gotypes.Int] = LayoutPair{Size: word, Align: word}
	out[gotypes.Uint] = LayoutPair{Size: word, Align: word}
	out[gotypes.Uintptr] = LayoutPair{Size: ptr, Align: ptr}
	out[gotypes.UnsafePointer] = LayoutPair{Size: ptr, Align: ptr}
	return out
}
