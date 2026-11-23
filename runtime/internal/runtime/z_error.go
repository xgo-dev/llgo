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

package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

type errorString string

func (e errorString) RuntimeError() {}

func (e errorString) Error() string {
	return "runtime error: " + string(e)
}

type plainError string

func (e plainError) Error() string {
	return string(e)
}

type typeAssertionErrorString string

func (e typeAssertionErrorString) RuntimeError() {}

func (e typeAssertionErrorString) Error() string {
	return string(e)
}

func AssertRuntimeError(b bool, msg string) {
	if b {
		panic(errorString(msg))
	}
}

func AssertNegativeShift(b bool) {
	if b {
		panic(errorString("negative shift amount"))
	}
}

func AssertIndexRange(b bool) {
	if b {
		panic(errorString("index out of range"))
	}
}

func PanicErrorString(msg string) {
	panic(errorString(msg))
}

func PanicIndex(x int, y int) {
	panic(boundsError{x: int64(x), signed: true, y: y, code: boundsIndex})
}

func PanicIndexU(x uint, y int) {
	panic(boundsError{x: int64(x), signed: false, y: y, code: boundsIndex})
}

func AssertDivideByZero(b bool) {
	if b {
		panic(errorString("integer divide by zero"))
	}
}

//go:noinline
func AssertNilDeref(b bool) {
	if b {
		panic(errorString("invalid memory address or nil pointer dereference"))
	}
}

func AssertNilDerefPtr(ptr unsafe.Pointer) unsafe.Pointer {
	AssertNilDeref(ptr == nil)
	return ptr
}

func PanicWrapNilPointer(b bool, recvType, methodName string) {
	if b {
		panic(plainError(abi.NilReceiverPanicMessage(recvType, methodName)))
	}
}

func PanicWrapNilPointerText(b bool, text string) {
	if b {
		panic(plainError(text))
	}
}

func PanicTypeAssertionError(msg string) {
	panic(typeAssertionErrorString(msg))
}

// printany prints a panic argument, including through Error and String methods.
// Keep the floating-point type assertions so their descriptors are present
// when the linker resolves the floating-point visibility predicates.
func printany(i any) {
	e := efaceOf(&i)
	typ := e._type
	if typ == nil {
		print("nil")
		return
	}
	switch v := i.(type) {
	case float32:
		print(v)
		return
	case float64:
		print(v)
		return
	case complex64:
		print(v)
		return
	case complex128:
		print(v)
		return
	case error:
		print(v.Error())
		return
	case interface{ String() string }:
		print(v.String())
		return
	}

	named := typ.TFlag&abi.TFlagUncommon != 0 // Predeclared scalars lack uncommon data.
	kind := typ.Kind()
	switch kind {
	case abi.String:
		s := *(*string)(e.data)
		if named {
			print(typ.Str_, `("`, s, `")`)
		} else {
			print(s)
		}
		return
	case abi.Complex64:
		if interfaceKindUsed(abi.Complex64) {
			println(typ.Str_, *(*complex64)(e.data))
		}
		return
	case abi.Complex128:
		if interfaceKindUsed(abi.Complex128) {
			println(typ.Str_, *(*complex128)(e.data))
		}
		return
	default:
		if kind > abi.Float64 {
			print("(", typ.String(), ") ", e.data)
			return
		}
	}

	if named {
		// Scalar descriptors never use the abbreviated name for pointers.
		print(typ.Str_, "(")
	}
	switch kind {
	case abi.Bool:
		print(*(*bool)(e.data))
	case abi.Float32:
		if interfaceKindUsed(abi.Float32) {
			print(*(*float32)(e.data))
		}
	case abi.Float64:
		if interfaceKindUsed(abi.Float64) {
			print(*(*float64)(e.data))
		}
	default:
		printPanicInteger(e.data, typ.Size_, kind <= abi.Int64)
	}
	if named {
		print(")")
	}
}

func efaceOf(ep *any) *eface {
	return (*eface)(unsafe.Pointer(ep))
}

// interfaceKindUsed includes named values of the requested underlying kind.
//
//go:linkname interfaceKindUsed llgo.interfaceKindUsed
func interfaceKindUsed(kind abi.Kind) bool

func printPanicInteger(data unsafe.Pointer, size uintptr, signed bool) {
	var u uint64
	switch size {
	case 1:
		u = uint64(*(*uint8)(data))
	case 2:
		u = uint64(*(*uint16)(data))
	case 4:
		u = uint64(*(*uint32)(data))
	default:
		u = *(*uint64)(data)
	}
	// An integer box is exactly the size of its dynamic type. Sign extend
	// from that width so narrow named integers keep their sign.
	negative := false
	if signed {
		shift := 64 - size*8
		v := int64(u<<shift) >> shift
		negative = v < 0
		if negative {
			u = uint64(-v)
		}
	}
	printDecimal(u, negative)
}

// PanicSIMDImmediate reports an invalid SIMD lane immediate.
func PanicSIMDImmediate() {
	panic(errorString("out-of-range immediate for simd intrinsic"))
}

// PanicSIMDUnimplemented identifies an intrinsic without an LLGo implementation.
func PanicSIMDUnimplemented(name string) {
	panic(errorString("unimplemented SIMD intrinsic: " + name))
}
