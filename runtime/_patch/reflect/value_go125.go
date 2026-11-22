//go:build go1.25
// +build go1.25

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

package reflect

import (
	"internal/abi"
	"unsafe"
)

// TypeAssert returns v's value as T and whether the assertion succeeded.
func TypeAssert[T any](v Value) (T, bool) {
	if v.flag == 0 {
		panic(&ValueError{"reflect.TypeAssert", Invalid})
	}
	if v.flag&flagRO != 0 {
		panic("reflect.TypeAssert: cannot return value obtained from unexported field or method")
	}
	v.mustNotPointToPackedCallback()
	if v.flag&flagMethod != 0 {
		v = makeMethodValue("TypeAssert", v)
	}
	if v.kind() == Interface {
		v = v.Elem()
		if !v.IsValid() {
			var zero T
			return zero, false
		}
	}

	typ := abi.TypeFor[T]()
	if typ.Kind() == abi.Interface {
		// Check compatibility before packing so a failed assertion does not allocate.
		iface := *(*any)(unsafe.Pointer(&abi.EmptyInterface{Type: v.typ()}))
		if out, ok := iface.(T); ok {
			// Empty and non-empty interfaces share the second data word.
			(*abi.EmptyInterface)(unsafe.Pointer(&out)).Data = packEfaceData(v)
			return out, true
		}
		var zero T
		return zero, false
	}
	if typ != v.typ() {
		v = assertFunc(v, typ)
		if !v.IsValid() {
			var zero T
			return zero, false
		}
	}

	v = v.goValue()
	if v.flag&flagIndir == 0 {
		return *(*T)(unsafe.Pointer(&v.ptr)), true
	}
	return *(*T)(v.ptr), true
}

// Implemented by LLGo's reflect replacement for its physical value layout.
func (v Value) goValue() Value
func (v Value) mustNotPointToPackedCallback()
func assertFunc(v Value, typ *abi.Type) Value
func packEfaceData(v Value) unsafe.Pointer
