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

// goIsHostLayoutMarker reports whether goT is structs.HostLayout (including
// aliases and defined types of structs.HostLayout). HostLayout deliberately
// does not propagate across nested struct boundaries.
func goIsHostLayoutMarker(goT gotypes.Type) bool {
	if goT == nil {
		return false
	}
	// Standard Go structs.HostLayout is `struct { _ hostLayout }` where
	// `hostLayout` is an unexported empty struct in package `structs`.
	if st, ok := goT.Underlying().(*gotypes.Struct); ok && st.NumFields() == 1 && st.Field(0).Name() == "_" {
		if n, ok := unaliasType(st.Field(0).Type()).(*gotypes.Named); ok && n.Obj().Pkg() != nil &&
			isStructsPkgPath(n.Obj().Pkg().Path()) &&
			n.Obj().Name() == "hostLayout" {
			if body, ok := n.Underlying().(*gotypes.Struct); ok && body.NumFields() == 0 {
				return true
			}
		}
	}
	// Also accept synthetic test markers named structs.HostLayout directly.
	if n, ok := unaliasType(goT).(*gotypes.Named); ok && n.Obj().Pkg() != nil &&
		isStructsPkgPath(n.Obj().Pkg().Path()) &&
		n.Obj().Name() == "HostLayout" {
		return true
	}
	return false
}

// goHasHostLayout reports whether goT's underlying struct has a direct
// structs.HostLayout marker field.
func goHasHostLayout(goT gotypes.Type) bool {
	if goT == nil {
		return false
	}
	st, ok := goT.Underlying().(*gotypes.Struct)
	if !ok {
		return false
	}
	for i := 0; i < st.NumFields(); i++ {
		if goIsHostLayoutMarker(st.Field(i).Type()) {
			return true
		}
	}
	return false
}

// goContainsHostLayout reports whether goT is or transitively contains a
// structs.HostLayout struct through fields or array elements.
func goContainsHostLayout(goT gotypes.Type) bool {
	if goT == nil {
		return false
	}
	switch u := goT.Underlying().(type) {
	case *gotypes.Struct:
		if goHasHostLayout(u) {
			return true
		}
		for i := 0; i < u.NumFields(); i++ {
			if goContainsHostLayout(u.Field(i).Type()) {
				return true
			}
		}
	case *gotypes.Array:
		return goContainsHostLayout(u.Elem())
	}
	return false
}

// goSIMDTagIndex recognizes simd/archsimd zero-sized vector alignment markers
// (v128 -> 0, v256 -> 1, v512 -> 2).
func goSIMDTagIndex(t gotypes.Type) (int, bool) {
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

// goSIMDVectorShape validates numeric and mask SIMD128/256/512 storage in
// simd/archsimd and returns its underlying lane array.
func goSIMDVectorShape(typ gotypes.Type) (*gotypes.Array, bool) {
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
	if named.Obj().Name() != expected {
		return nil, false
	}
	var wantTag int
	switch bits * lanes.Len() {
	case 128:
		wantTag = 0
	case 256:
		wantTag = 1
	case 512:
		wantTag = 2
	default:
		return nil, false
	}
	tag, ok := goSIMDTagIndex(st.Field(0).Type())
	if !ok || tag != wantTag {
		return nil, false
	}
	return lanes, true
}

// DefaultSIMDAlignments returns target-accurate vector alignments for
// v128, v256, and v512 without requiring LLVM TargetData.
func DefaultSIMDAlignments(spec target.Spec) [3]int64 {
	if spec.WordSize == 0 || spec.PtrSize == 0 {
		spec = target.ResolveSpec(spec)
	}
	// Clang caps fixed-length AArch64 C vector alignment at 128 bits.
	if spec.GOARCH == "arm64" {
		return [3]int64{16, 16, 16}
	}
	return [3]int64{16, 32, 64}
}

// DefaultHostBasicLayouts computes standard C ABI (size, align) for Go scalar
// basic kinds on spec without requiring LLVM TargetData.
func DefaultHostBasicLayouts(spec target.Spec) [UnsafePointer + 1]LayoutPair {
	if spec.WordSize == 0 || spec.PtrSize == 0 {
		spec = target.ResolveSpec(spec)
	}
	word := int64(spec.WordSize)
	ptr := int64(spec.PtrSize)
	i64Align := int64(8)
	if spec.GOARCH == "386" && spec.GOOS != "windows" {
		// System V i386 ABI aligns 64-bit integers and float64 to 4 bytes.
		i64Align = 4
	}
	return [UnsafePointer + 1]LayoutPair{
		Bool:          {Size: 1, Align: 1},
		Int:           {Size: word, Align: word},
		Int8:          {Size: 1, Align: 1},
		Int16:         {Size: 2, Align: 2},
		Int32:         {Size: 4, Align: 4},
		Int64:         {Size: 8, Align: i64Align},
		Uint:          {Size: word, Align: word},
		Uint8:         {Size: 1, Align: 1},
		Uint16:        {Size: 2, Align: 2},
		Uint32:        {Size: 4, Align: 4},
		Uint64:        {Size: 8, Align: i64Align},
		Uintptr:       {Size: ptr, Align: ptr},
		Float32:       {Size: 4, Align: 4},
		Float64:       {Size: 8, Align: i64Align},
		Complex64:     {Size: 8, Align: 4},
		Complex128:    {Size: 16, Align: i64Align},
		UnsafePointer: {Size: ptr, Align: ptr},
	}
}
