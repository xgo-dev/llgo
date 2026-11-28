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

// DataKind classifies how a concrete value is stored in an interface data word.
type DataKind int

const (
	DataInvalid  DataKind = iota
	DataIndirect          // allocate memory and store pointer in interface word
	DataPointer           // store pointer value directly in interface word
	DataInteger           // store integer value directly in interface word
	DataBitCast           // store bitcast scalar (e.g. float) directly in interface word
)

// DataKindOf classifies how an LLGo Type fits in an interface data word,
// unwrapping single-field structs and single-element arrays while returning the
// inner Type and unwrap depth.
func DataKindOf(t Type, is32Bits bool) (DataKind, Type, int) {
	return dataKindOf(t, 0, is32Bits)
}

func dataKindOf(t Type, lvl int, is32Bits bool) (DataKind, Type, int) {
	if t == nil {
		return DataInvalid, nil, lvl
	}
	if a, ok := t.(*Alias); ok && a.hasCallConv && a.callConv.IsNative() {
		if _, isSig := a.Underlying().(*Signature); isSig {
			return DataPointer, t, lvl
		}
	}
	if n, ok := t.(*Named); ok && n.hasCallConv && n.callConv.IsNative() {
		if _, isSig := n.Underlying().(*Signature); isSig {
			return DataPointer, t, lvl
		}
	}
	switch u := t.Underlying().(type) {
	case *Basic:
		switch {
		case Bool <= u.kind && u.kind <= Uintptr:
			if is32Bits && (u.kind == Int64 || u.kind == Uint64) {
				return DataIndirect, t, lvl
			}
			return DataInteger, t, lvl
		case u.kind == Float32:
			return DataBitCast, t, lvl
		case u.kind == Float64:
			if is32Bits {
				return DataIndirect, t, lvl
			}
			return DataBitCast, t, lvl
		case u.kind == UnsafePointer:
			return DataPointer, t, lvl
		}
	case *Pointer, *Map, *Chan:
		return DataPointer, t, lvl
	case *Signature:
		if u.callConv.IsNative() {
			return DataPointer, t, lvl
		}
		return DataIndirect, t, lvl
	case *Struct:
		if len(u.fields) == 1 {
			return dataKindOf(u.fields[0].typ, lvl+1, is32Bits)
		}
	case *Array:
		if u.length == 1 {
			return dataKindOf(u.elem, lvl+1, is32Bits)
		}
	}
	return DataIndirect, t, lvl
}

// gcBitmap encodes sorted GC-traced pointer byte offsets into Go's standard
// 1-bit-per-pointer-word GCData bitmap spanning [0, ptrBytes).
func gcBitmap(ptrOffsets []int64, ptrBytes, wordSize int64) []byte {
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

// EqualName returns the runtime equality helper symbol name for t, or "" if t
// is not comparable.
func (c *Context) EqualName(t Type) string {
	if t == nil {
		return ""
	}
	switch u := t.Underlying().(type) {
	case *Basic:
		switch u.kind {
		case Bool, Int8, Uint8:
			return "memequal8"
		case Int16, Uint16:
			return "memequal16"
		case Int32, Uint32:
			return "memequal32"
		case Int64, Uint64:
			return "memequal64"
		case Int, Uint, Uintptr:
			if c.spec.WordSize == 8 {
				return "memequal64"
			}
			return "memequal32"
		case Float32:
			return "f32equal"
		case Float64:
			return "f64equal"
		case Complex64:
			return "c64equal"
		case Complex128:
			return "c128equal"
		case String:
			return "strequal"
		case UnsafePointer:
			return "memequalptr"
		}
	case *Pointer, *Chan:
		return "memequalptr"
	case *Slice, *Signature, *Map:
		return ""
	case *Interface:
		if u.Empty() {
			return "nilinterequal"
		}
		return "interequal"
	case *Struct:
		if len(u.fields) == 0 {
			return "memequal0"
		}
		for _, f := range u.fields {
			if c.EqualName(f.typ) == "" {
				return ""
			}
		}
		return "structequal"
	case *Array:
		if c.EqualName(u.elem) != "" {
			if c.Layout(u.elem).Size == 0 {
				return "memequal0"
			}
			return "arrayequal"
		}
		return ""
	case *SIMDVec:
		return "structequal"
	}
	return ""
}
