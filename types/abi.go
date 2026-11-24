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

import gotypes "go/types"

// DataKind classifies how a concrete value is stored in an interface data word.
type DataKind int

const (
	DataInvalid  DataKind = iota
	DataIndirect          // allocate memory and store pointer in interface word
	DataPointer           // store pointer value directly in interface word
	DataInteger           // store integer value directly in interface word
	DataBitCast           // store bitcast scalar (e.g. float) directly in interface word
)

// DataKindOf classifies how a physical LLGo Type fits in an interface data word.
func DataKindOf(t Type, is32Bits bool) DataKind {
	if t == nil {
		return DataInvalid
	}
	switch u := t.Underlying().(type) {
	case *Basic:
		switch {
		case gotypes.Bool <= u.Kind && u.Kind <= gotypes.Uintptr:
			if is32Bits && (u.Kind == gotypes.Int64 || u.Kind == gotypes.Uint64) {
				return DataIndirect
			}
			return DataInteger
		case u.Kind == gotypes.Float32:
			return DataBitCast
		case u.Kind == gotypes.Float64:
			if is32Bits {
				return DataIndirect
			}
			return DataBitCast
		case u.Kind == gotypes.UnsafePointer:
			return DataPointer
		}
	case *Pointer, *Func, *Map, *Chan:
		return DataPointer
	case *Struct:
		if len(u.Fields) == 1 {
			return DataKindOf(u.Fields[0].Type, is32Bits)
		}
	case *Array:
		if u.Len == 1 {
			return DataKindOf(u.Elem, is32Bits)
		}
	}
	return DataIndirect
}

// GoDataKindOf classifies how a go/types.Type fits in an interface data word,
// unwrapping single-field structs and single-element arrays while tracking
// unwrap depth.
func GoDataKindOf(raw gotypes.Type, lvl int, is32Bits bool) (DataKind, gotypes.Type, int) {
	switch t := raw.Underlying().(type) {
	case *gotypes.Basic:
		kind := t.Kind()
		switch {
		case gotypes.Bool <= kind && kind <= gotypes.Uintptr:
			if is32Bits && (kind == gotypes.Int64 || kind == gotypes.Uint64) {
				return DataIndirect, raw, lvl
			}
			return DataInteger, raw, lvl
		case kind == gotypes.Float32:
			return DataBitCast, raw, lvl
		case kind == gotypes.Float64:
			if is32Bits {
				return DataIndirect, raw, lvl
			}
			return DataBitCast, raw, lvl
		case kind == gotypes.UnsafePointer:
			return DataPointer, raw, lvl
		}
	case *gotypes.Pointer, *gotypes.Signature, *gotypes.Map, *gotypes.Chan:
		return DataPointer, raw, lvl
	case *gotypes.Struct:
		if t.NumFields() == 1 {
			return GoDataKindOf(t.Field(0).Type(), lvl+1, is32Bits)
		}
	case *gotypes.Interface, *gotypes.Slice:
	case *gotypes.Array:
		if t.Len() == 1 {
			return GoDataKindOf(t.Elem(), lvl+1, is32Bits)
		}
	default:
		panic("unknown type")
	}
	return DataIndirect, raw, lvl
}

// GCBitmap encodes sorted GC-traced pointer byte offsets into Go's standard
// 1-bit-per-pointer-word GCData bitmap spanning [0, ptrBytes).
func GCBitmap(ptrOffsets []int64, ptrBytes, wordSize int64) []byte {
	if ptrBytes <= 0 || wordSize <= 0 || len(ptrOffsets) == 0 {
		return nil
	}
	words := (ptrBytes + wordSize - 1) / wordSize
	bitmap := make([]byte, (words+7)/8)
	for _, off := range ptrOffsets {
		if off < 0 || off >= ptrBytes {
			continue
		}
		wordIdx := off / wordSize
		bitmap[wordIdx/8] |= 1 << uint(wordIdx%8)
	}
	return bitmap
}

// GoHasPointers reports whether t contains any GC-traced pointer words without
// computing field offsets or array sizes.
func GoHasPointers(t gotypes.Type) bool {
	switch u := unaliasType(t).(type) {
	case *gotypes.Basic:
		return u.Kind() == gotypes.String || u.Kind() == gotypes.UnsafePointer
	case *gotypes.Pointer, *gotypes.Slice, *gotypes.Signature, *gotypes.Interface, *gotypes.Map, *gotypes.Chan:
		return true
	case *gotypes.Struct:
		for i := 0; i < u.NumFields(); i++ {
			if GoHasPointers(u.Field(i).Type()) {
				return true
			}
		}
		return false
	case *gotypes.Array:
		return u.Len() != 0 && GoHasPointers(u.Elem())
	case *gotypes.Named:
		return GoHasPointers(u.Underlying())
	}
	panic("unsupported pointer layout: " + t.String())
}

// GoPtrBytes computes the prefix size in bytes containing all GC-traced
// pointers in t (matching Go runtime/abi.Type.PtrBytes).
func GoPtrBytes(t gotypes.Type, ptrSize int64, sizes gotypes.Sizes) int64 {
	switch u := unaliasType(t).(type) {
	case *gotypes.Basic:
		switch u.Kind() {
		case gotypes.String, gotypes.UnsafePointer:
			return ptrSize
		}
		return 0
	case *gotypes.Pointer, *gotypes.Slice, *gotypes.Signature, *gotypes.Map, *gotypes.Chan:
		return ptrSize
	case *gotypes.Interface:
		return 2 * ptrSize
	case *gotypes.Struct:
		lastField := -1
		var lastBytes int64
		n := u.NumFields()
		fields := make([]*gotypes.Var, n)
		for i := 0; i < n; i++ {
			f := u.Field(i)
			fields[i] = f
			if pb := GoPtrBytes(f.Type(), ptrSize, sizes); pb != 0 {
				lastField = i
				lastBytes = pb
			}
		}
		if lastField == -1 {
			return 0
		}
		return sizes.Offsetsof(fields)[lastField] + lastBytes
	case *gotypes.Array:
		if u.Len() != 0 {
			elem := u.Elem()
			if pb := GoPtrBytes(elem, ptrSize, sizes); pb != 0 {
				return goABISize(u, ptrSize, sizes) - goABISize(elem, ptrSize, sizes) + pb
			}
		}
		return 0
	case *gotypes.Named:
		return GoPtrBytes(u.Underlying(), ptrSize, sizes)
	}
	panic("unsupported ptrbytes: " + t.String())
}

func goABISize(t gotypes.Type, ptrSize int64, sizes gotypes.Sizes) int64 {
	switch u := unaliasType(t).(type) {
	case *gotypes.Basic:
		switch u.Kind() {
		case gotypes.Bool, gotypes.Int8, gotypes.Uint8:
			return 1
		case gotypes.Int16, gotypes.Uint16:
			return 2
		case gotypes.Int32, gotypes.Uint32, gotypes.Float32:
			return 4
		case gotypes.Int64, gotypes.Uint64, gotypes.Float64, gotypes.Complex64:
			return 8
		case gotypes.Complex128:
			return 16
		case gotypes.Int, gotypes.Uint, gotypes.Uintptr, gotypes.UnsafePointer:
			return ptrSize
		case gotypes.String:
			return 2 * ptrSize
		}
	case *gotypes.Pointer, *gotypes.Signature, *gotypes.Map, *gotypes.Chan:
		return ptrSize
	case *gotypes.Slice:
		return 3 * ptrSize
	case *gotypes.Interface:
		return 2 * ptrSize
	case *gotypes.Struct:
		return sizes.Sizeof(u)
	case *gotypes.Array:
		return u.Len() * goABISize(u.Elem(), ptrSize, sizes)
	case *gotypes.Named:
		return goABISize(u.Underlying(), ptrSize, sizes)
	}
	panic("unsupported size: " + t.String())
}

// GoIsRegularMemory reports whether values of t may be compared and hashed
// bytewise (no floats, strings, slices, funcs, interfaces, maps, padding bytes,
// or blank struct fields).
func GoIsRegularMemory(t gotypes.Type, ptrSize int64, sizes gotypes.Sizes) bool {
	switch u := unaliasType(t).(type) {
	case *gotypes.Basic:
		switch u.Kind() {
		case gotypes.Float32, gotypes.Float64, gotypes.Complex64, gotypes.Complex128, gotypes.String:
			return false
		}
		return true
	case *gotypes.Pointer, *gotypes.Chan:
		return true
	case *gotypes.Slice, *gotypes.Signature, *gotypes.Interface, *gotypes.Map:
		return false
	case *gotypes.Struct:
		switch n := u.NumFields(); n {
		case 0:
			return true
		case 1:
			f := u.Field(0)
			if f.Name() == "_" {
				return false
			}
			return GoIsRegularMemory(f.Type(), ptrSize, sizes)
		default:
			fields := make([]*gotypes.Var, n)
			for i := 0; i < n; i++ {
				fields[i] = u.Field(i)
			}
			offsets := sizes.Offsetsof(fields)
			end := goABISize(u, ptrSize, sizes)
			for i := 0; i < n; i++ {
				f := fields[i]
				fieldEnd := end
				if i+1 < len(offsets) {
					fieldEnd = offsets[i+1]
				}
				padded := offsets[i]+goABISize(f.Type(), ptrSize, sizes) != fieldEnd
				if f.Name() == "_" || !GoIsRegularMemory(f.Type(), ptrSize, sizes) || padded {
					return false
				}
			}
			return true
		}
	case *gotypes.Array:
		if GoIsRegularMemory(u.Elem(), ptrSize, sizes) {
			return true
		}
		return u.Len() == 0
	case *gotypes.Named:
		return GoIsRegularMemory(u.Underlying(), ptrSize, sizes)
	}
	return true
}

// GoEqualName returns the runtime equality helper symbol name for t, or "" if
// t is not comparable.
func GoEqualName(t gotypes.Type, ptrSize int64, sizes gotypes.Sizes) string {
	switch u := unaliasType(t).(type) {
	case *gotypes.Basic:
		switch u.Kind() {
		case gotypes.Bool, gotypes.Int8, gotypes.Uint8:
			return "memequal8"
		case gotypes.Int16, gotypes.Uint16:
			return "memequal16"
		case gotypes.Int32, gotypes.Uint32:
			return "memequal32"
		case gotypes.Int64, gotypes.Uint64:
			return "memequal64"
		case gotypes.Int, gotypes.Uint, gotypes.Uintptr:
			if ptrSize == 8 {
				return "memequal64"
			}
			return "memequal32"
		case gotypes.Float32:
			return "f32equal"
		case gotypes.Float64:
			return "f64equal"
		case gotypes.Complex64:
			return "c64equal"
		case gotypes.Complex128:
			return "c128equal"
		case gotypes.String:
			return "strequal"
		case gotypes.UnsafePointer:
			return "memequalptr"
		}
	case *gotypes.Pointer, *gotypes.Chan:
		return "memequalptr"
	case *gotypes.Slice, *gotypes.Signature, *gotypes.Map:
		return ""
	case *gotypes.Interface:
		if u.NumMethods() == 0 {
			return "nilinterequal"
		}
		return "interequal"
	case *gotypes.Struct:
		n := u.NumFields()
		if n == 0 {
			return "memequal0"
		}
		for i := 0; i < n; i++ {
			if GoEqualName(u.Field(i).Type(), ptrSize, sizes) == "" {
				return ""
			}
		}
		return "structequal"
	case *gotypes.Array:
		elem := u.Elem()
		if GoEqualName(elem, ptrSize, sizes) != "" {
			if goABISize(elem, ptrSize, sizes) == 0 {
				return "memequal0"
			}
			return "arrayequal"
		}
		return ""
	case *gotypes.Named:
		return GoEqualName(u.Underlying(), ptrSize, sizes)
	}
	panic("unsupported equal: " + t.String())
}
