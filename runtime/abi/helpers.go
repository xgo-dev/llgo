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

import "unsafe"

func NoEscape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}

type EmptyInterface struct {
	Type *Type
	Data unsafe.Pointer
}

// TypeOf returns the runtime Type of some value.
func TypeOf(a any) *Type {
	eface := *(*EmptyInterface)(unsafe.Pointer(&a))
	return (*Type)(NoEscape(unsafe.Pointer(eface.Type)))
}

// TypeFor returns the runtime Type for a type parameter.
func TypeFor[T any]() *Type {
	var v T
	if t := TypeOf(v); t != nil {
		return t
	}
	return TypeOf((*T)(nil)).Elem()
}

// EscapeNonString is modeled by the compiler.
func EscapeNonString[T any](v T) {
}

// EscapeToResultNonString models a data-flow edge to the result.
func EscapeToResultNonString[T any](v T) T {
	EscapeNonString(v)
	return *(*T)(NoEscape(unsafe.Pointer(&v)))
}
