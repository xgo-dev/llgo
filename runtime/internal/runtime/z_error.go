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

	"github.com/goplus/llgo/runtime/abi"
	"github.com/goplus/llgo/runtime/internal/clite/bitcast"
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

func AssertDivideByZero(b bool) {
	if b {
		panic(errorString("integer divide by zero"))
	}
}

func AssertNilDeref(b bool) {
	if b {
		panic(errorString("invalid memory address or nil pointer dereference"))
	}
}

func PanicWrapNilPointer(b bool, recvType, methodName string) {
	if b {
		recvType = panicWrapRecvType(recvType)
		panic(plainError("value method " + recvType + "." + methodName + " called using nil *" + panicWrapTypeName(recvType) + " pointer"))
	}
}

func panicWrapRecvType(recvType string) string {
	const commandLineArguments = "command-line-arguments."
	if len(recvType) > len(commandLineArguments) && recvType[:len(commandLineArguments)] == commandLineArguments {
		return "main." + recvType[len(commandLineArguments):]
	}
	return recvType
}

func panicWrapTypeName(recvType string) string {
	depth := 0
	for i := len(recvType) - 1; i >= 0; i-- {
		switch recvType[i] {
		case ']':
			depth++
		case '[':
			if depth > 0 {
				depth--
			}
		case '.':
			if depth == 0 {
				return recvType[i+1:]
			}
		}
	}
	return recvType
}

func PanicTypeAssertionError(msg string) {
	panic(typeAssertionErrorString(msg))
}

// printany prints an argument passed to panic.
// If panic is called with a value that has a String or Error method,
// it has already been converted into a string by preprintpanics.
func printany(i any) {
	switch v := i.(type) {
	case nil:
		print("nil")
	case bool:
		print(v)
	case int:
		print(v)
	case int8:
		print(v)
	case int16:
		print(v)
	case int32:
		print(v)
	case int64:
		print(v)
	case uint:
		print(v)
	case uint8:
		print(v)
	case uint16:
		print(v)
	case uint32:
		print(v)
	case uint64:
		print(v)
	case uintptr:
		print(v)
	case float32:
		print(v)
	case float64:
		print(v)
	case complex64:
		print(v)
	case complex128:
		print(v)
	case string:
		print(v)
	case error:
		print(v.Error())
	case interface{ String() string }:
		print(v.String())
	default:
		printanycustomtype(i)
	}
}

func efaceOf(ep *any) *eface {
	return (*eface)(unsafe.Pointer(ep))
}

func printanycustomtype(i any) {
	e := efaceOf(&i)
	typestring := e._type.String()

	switch e._type.Kind() {
	case abi.String:
		print(typestring, `("`, *(*string)(e.data), `")`)
	case abi.Bool:
		if isDirectIface(e._type) {
			print(typestring, "(", uintptr(e.data) != 0, ")")
		} else {
			print(typestring, "(", *(*bool)(e.data), ")")
		}
	case abi.Int, abi.Int8, abi.Int16, abi.Int32, abi.Int64:
		if isDirectIface(e._type) {
			print(typestring, "(", int64(uintptr(e.data)), ")")
		} else {
			print(typestring, "(", *(*int64)(e.data), ")")
		}
	case abi.Uint, abi.Uint8, abi.Uint16, abi.Uint32, abi.Uint64, abi.Uintptr:
		if isDirectIface(e._type) {
			print(typestring, "(", uint64(uintptr(e.data)), ")")
		} else {
			print(typestring, "(", *(*uint64)(e.data), ")")
		}
	case abi.Float32:
		if isDirectIface(e._type) {
			print(typestring, "(", bitcast.ToFloat32(int32(uintptr(e.data))), ")")
		} else {
			print(typestring, "(", *(*float32)(e.data), ")")
		}
	case abi.Float64:
		if isDirectIface(e._type) {
			print(typestring, "(", bitcast.ToFloat64(int64(uintptr(e.data))), ")")
		} else {
			print(typestring, "(", *(*float64)(e.data), ")")
		}
	case abi.Complex64:
		println(typestring, *(*complex64)(e.data))
	case abi.Complex128:
		println(typestring, *(*complex128)(e.data))
	default:
		print("(", typestring, ") ", e.data)
	}
}
