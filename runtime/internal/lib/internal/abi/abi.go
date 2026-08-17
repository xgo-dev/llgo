/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package abi

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

type InterfaceType = abi.InterfaceType

func NoEscape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}

//go:linkname FuncPCABI0 llgo.funcPCABI0
func FuncPCABI0(f interface{}) uintptr

func FuncPCABIInternal(f interface{}) uintptr {
	return FuncPCABI0(f)
}

type Type = abi.Type

type EmptyInterface struct {
	Type *Type
	Data unsafe.Pointer
}

// TypeOf returns the abi.Type of some value.
func TypeOf(a any) *Type {
	eface := *(*EmptyInterface)(unsafe.Pointer(&a))
	// Types are either static (for compiler-created types) or
	// heap-allocated but always reachable (for reflection-created
	// types, held in the central map). So there is no need to
	// escape types. noescape here help avoid unnecessary escape
	// of v.
	return (*Type)(NoEscape(unsafe.Pointer(eface.Type)))
}

// TypeFor returns the abi.Type for a type parameter.
func TypeFor[T any]() *Type {
	var v T
	if t := TypeOf(v); t != nil {
		return t // optimize for T being a non-interface kind
	}
	return TypeOf((*T)(nil)).Elem() // only for an interface kind
}

// EscapeNonString forces v to be on the heap, if v contains a
// non-string pointer.
//
// This is used in hash/maphash.Comparable. We cannot hash pointers
// to local variables on stack, as their addresses might change on
// stack growth. Strings are okay as the hash depends on only the
// content, not the pointer.
//
// This is essentially
//
//	if hasNonStringPointers(T) { Escape(v) }
//
// Implemented as a compiler intrinsic.
func EscapeNonString[T any](v T) {
	//panic("intrinsic")
}

// EscapeToResultNonString models a data flow edge from v to the result,
// if v contains a non-string pointer. If v contains only string pointers,
// it returns a copy of v, but is not modeled as a data flow edge
// from the escape analysis's perspective.
//
// This is used in unique.clone, to model the data flow edge on the
// value with strings excluded, because strings are cloned (by
// content).
//
// TODO: probably we should define this as a intrinsic and EscapeNonString
// could just be "heap = EscapeToResultNonString(v)". This way we can model
// an edge to the result but not necessarily heap.
func EscapeToResultNonString[T any](v T) T {
	EscapeNonString(v)
	return *(*T)(NoEscape(unsafe.Pointer(&v)))
}
