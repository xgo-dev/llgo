// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js && wasm
// +build js,wasm

// Package js gives access to the WebAssembly host environment when using the js/wasm architecture.
// Its API is based on JavaScript semantics.
//
// This package is EXPERIMENTAL. Its current scope is only to allow tests to run, but not yet to provide a
// comprehensive API for users. It is exempt from the Go compatibility promise.
package js

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

//llgo:skipall
type _js struct{}

// Value represents a JavaScript value. The zero value is the JavaScript value "undefined".
// Values can be checked for equality with the Equal method.
type Value struct {
	ref uintptr
}

func floatValue(f float64) Value {
	if f == 0 {
		return valueZero
	}
	if f != f {
		return valueNaN
	}
	return emval_new_double(f)
}

func intValue(i int) Value {
	if i == 0 {
		return valueZero
	}
	return emval_new_double(float64(i))
}

// Error wraps a JavaScript error.
type Error struct {
	// Value is the underlying JavaScript error value.
	Value
}

// Error implements the error interface.
func (e Error) Error() string {
	return "JavaScript error: " + e.Get("message").String()
}

/*
var (
	valueUndefined = Value{ref: 0}
	valueNaN       = predefValue(0, typeFlagNone)
	valueZero      = predefValue(1, typeFlagNone)
	valueNull      = predefValue(2, typeFlagNone)
	valueTrue      = predefValue(3, typeFlagNone)
	valueFalse     = predefValue(4, typeFlagNone)
	valueGlobal    = predefValue(5, typeFlagObject)
	jsGo           = predefValue(6, typeFlagObject) // instance of the Go class in JavaScript

	objectConstructor = valueGlobal.Get("Object")
	arrayConstructor  = valueGlobal.Get("Array")
)
*/

// Equal reports whether v and w are equal according to JavaScript's === operator.
func (v Value) Equal(w Value) bool {
	return emval_equals(v, w) && v.ref != valueNaN.ref
}

// Undefined returns the JavaScript value "undefined".
func Undefined() Value {
	return valueUndefined
}

// IsUndefined reports whether v is the JavaScript value "undefined".
func (v Value) IsUndefined() bool {
	return v.ref == valueUndefined.ref
}

// Null returns the JavaScript value "null".
func Null() Value {
	return valueNull
}

// IsNull reports whether v is the JavaScript value "null".
func (v Value) IsNull() bool {
	return v.ref == valueNull.ref
}

// IsNaN reports whether v is the JavaScript value "NaN".
func (v Value) IsNaN() bool {
	return v.ref == valueNaN.ref
}

// Global returns the JavaScript global object, usually "window" or "global".
func Global() Value {
	return valueGlobal
}

// ValueOf returns x as a JavaScript value:
//
//	| Go                     | JavaScript             |
//	| ---------------------- | ---------------------- |
//	| js.Value               | [its value]            |
//	| js.Func                | function               |
//	| nil                    | null                   |
//	| bool                   | boolean                |
//	| integers and floats    | number                 |
//	| string                 | string                 |
//	| []interface{}          | new array              |
//	| map[string]interface{} | new object             |
//
// Panics if x is not one of the expected types.
func ValueOf(x any) Value {
	switch x := x.(type) {
	case Value:
		return x
	case Func:
		return x.Value
	case nil:
		return valueNull
	case bool:
		if x {
			return valueTrue
		} else {
			return valueFalse
		}
	case int:
		return floatValue(float64(x))
	case int8:
		return floatValue(float64(x))
	case int16:
		return floatValue(float64(x))
	case int32:
		return floatValue(float64(x))
	case int64:
		return floatValue(float64(x))
	case uint:
		return floatValue(float64(x))
	case uint8:
		return floatValue(float64(x))
	case uint16:
		return floatValue(float64(x))
	case uint32:
		return floatValue(float64(x))
	case uint64:
		return floatValue(float64(x))
	case uintptr:
		return floatValue(float64(x))
	case unsafe.Pointer:
		return floatValue(float64(uintptr(x)))
	case float32:
		return floatValue(float64(x))
	case float64:
		return floatValue(x)
	case string:
		return stringVal(x)
	case []any:
		a := arrayConstructor.New(len(x))
		for i, s := range x {
			a.SetIndex(i, s)
		}
		return a
	case map[string]any:
		o := objectConstructor.New()
		for k, v := range x {
			o.Set(k, v)
		}
		return o
	default:
		panic("ValueOf: invalid value")
	}
}

func stringVal(x string) Value {
	return emval_new_string(c.AllocaCStr(x))
}

// Type represents the JavaScript type of a Value.
type Type int

const (
	TypeUndefined Type = iota
	TypeNull
	TypeBoolean
	TypeNumber
	TypeString
	TypeSymbol
	TypeObject
	TypeFunction
)

func (t Type) String() string {
	switch t {
	case TypeUndefined:
		return "undefined"
	case TypeNull:
		return "null"
	case TypeBoolean:
		return "boolean"
	case TypeNumber:
		return "number"
	case TypeString:
		return "string"
	case TypeSymbol:
		return "symbol"
	case TypeObject:
		return "object"
	case TypeFunction:
		return "function"
	default:
		panic("bad type")
	}
}

func (t Type) isObject() bool {
	return t == TypeObject || t == TypeFunction
}

// Type returns the JavaScript type of the value v. It is similar to JavaScript's typeof operator,
// except that it returns TypeNull instead of TypeObject for null.
func (v Value) Type() Type {
	switch v.ref {
	case valueUndefined.ref:
		return TypeUndefined
	case valueNull.ref:
		return TypeNull
	case valueTrue.ref, valueFalse.ref:
		return TypeBoolean
	}
	if emval_is_number(v) {
		return TypeNumber
	} else if emval_is_string(v) {
		return TypeString
	}
	switch emval_as_string(emval_typeof(v)) {
	case "object":
		return TypeObject
	case "symbol":
		return TypeSymbol
	case "function":
		return TypeFunction
	default:
		panic("bad type")
	}
}

// Get returns the JavaScript property p of value v.
// It panics if v is not a JavaScript object.
func (v Value) Get(p string) Value {
	if vType := v.Type(); !vType.isObject() {
		panic(&ValueError{"Value.Get", vType})
	}
	return emval_get_property(v, stringVal(p))
	// r := makeValue(valueGet(v.ref, p))
	// runtime.KeepAlive(v)
	// return r
}

// Set sets the JavaScript property p of value v to ValueOf(x).
// It panics if v is not a JavaScript object.
func (v Value) Set(p string, x any) {
	if vType := v.Type(); !vType.isObject() {
		panic(&ValueError{"Value.Set", vType})
	}
	// xv := ValueOf(x)
	emval_set_property(v, stringVal(p), ValueOf(x))
	// valueSet(v.ref, p, xv.ref)
	// runtime.KeepAlive(v)
	// runtime.KeepAlive(xv)
}

// // valueSet sets property p of ref v to ref x.
// //
// // Using go:noescape is safe because no references are maintained to the
// // Go string p after the syscall returns.
// //
// //go:wasmimport gojs syscall/js.valueSet
// //go:noescape
// func valueSet(v ref, p string, x ref)

// Delete deletes the JavaScript property p of value v.
// It panics if v is not a JavaScript object.
func (v Value) Delete(p string) {
	if vType := v.Type(); !vType.isObject() {
		panic(&ValueError{"Value.Delete", vType})
	}
	emval_delete(v, ValueOf(p))
	//valueDelete(v.ref, p)
	//runtime.KeepAlive(v)
}

// // valueDelete deletes the JavaScript property p of ref v.
// //
// // Using go:noescape is safe because no references are maintained to the
// // Go string p after the syscall returns.
// //
// //go:wasmimport gojs syscall/js.valueDelete
// //go:noescape
// func valueDelete(v ref, p string)

// Index returns JavaScript index i of value v.
// It panics if v is not a JavaScript object.
func (v Value) Index(i int) Value {
	if vType := v.Type(); !vType.isObject() {
		panic(&ValueError{"Value.Index", vType})
	}
	return emval_get_property(v, intValue(i))
	// r := makeValue(valueIndex(v.ref, i))
	// runtime.KeepAlive(v)
	// return r
}

// //go:wasmimport gojs syscall/js.valueIndex
// func valueIndex(v ref, i int) ref

// SetIndex sets the JavaScript index i of value v to ValueOf(x).
// It panics if v is not a JavaScript object.
func (v Value) SetIndex(i int, x any) {
	if vType := v.Type(); !vType.isObject() {
		panic(&ValueError{"Value.SetIndex", vType})
	}
	emval_set_property(v, intValue(i), ValueOf(x))
	// xv := ValueOf(x)
	// valueSetIndex(v.ref, i, xv.ref)
	// runtime.KeepAlive(v)
	// runtime.KeepAlive(xv)
}

// //go:wasmimport gojs syscall/js.valueSetIndex
// func valueSetIndex(v ref, i int, x ref)

// makeArgSlices makes two slices to hold JavaScript arg data.
// It can be paired with storeArgs to make-and-store JavaScript arg slices.
// However, the two functions are separated to ensure makeArgSlices is inlined
// which will prevent the slices from being heap allocated for small (<=16)
// numbers of args.
// func makeArgSlices(size int) (argVals []Value, argRefs []ref) {
// 	// value chosen for being power of two, and enough to handle all web APIs
// 	// in particular, note that WebGL2's texImage2D takes up to 10 arguments
// 	const maxStackArgs = 16
// 	if size <= maxStackArgs {
// 		// as long as makeArgs is inlined, these will be stack-allocated
// 		argVals = make([]Value, size, maxStackArgs)
// 		argRefs = make([]ref, size, maxStackArgs)
// 	} else {
// 		// allocates on the heap, but exceeding maxStackArgs should be rare
// 		argVals = make([]Value, size)
// 		argRefs = make([]ref, size)
// 	}
// 	return
// }

// storeArgs maps input args onto respective Value and ref slices.
// It can be paired with makeArgSlices to make-and-store JavaScript arg slices.
// func storeArgs(args []any, argValsDst []Value, argRefsDst []ref) {
// 	// would go in makeArgs if the combined func was simple enough to inline
// 	for i, arg := range args {
// 		v := ValueOf(arg)
// 		argValsDst[i] = v
// 		argRefsDst[i] = v.ref
// 	}
// }

// Length returns the JavaScript property "length" of v.
// It panics if v is not a JavaScript object.
func (v Value) Length() int {
	if vType := v.Type(); !vType.isObject() {
		panic(&ValueError{"Value.SetIndex", vType})
	}
	return emval_get_property(v, emval_new_string(c.Str("length"))).Int()
	//r := valueLength(v.ref)
	//runtime.KeepAlive(v)
	//return r
}

// //go:wasmimport gojs syscall/js.valueLength
// func valueLength(v ref) int

// Call does a JavaScript call to the method m of value v with the given arguments.
// It panics if v has no method m.
// The arguments get mapped to JavaScript values according to the ValueOf function.
func (v Value) Call(m string, args ...any) (res Value) {
	var err c.Int
	if len(args) == 0 {
		res = emval_method_call(v, c.AllocaCStr(m), nil, 0, &err)
	} else {
		vargs := make([]Value, len(args))
		for i := 0; i < len(args); i++ {
			vargs[i] = ValueOf(args[i])
		}
		res = emval_method_call(v, c.AllocaCStr(m), *(**Value)(unsafe.Pointer(&vargs)), c.Int(len(args)), &err)
	}
	if err != 0 {
		if vType := v.Type(); !vType.isObject() { // check here to avoid overhead in success case
			panic(&ValueError{"Value.Call", vType})
		}
		if propType := v.Get(m).Type(); propType != TypeFunction {
			panic("syscall/js: Value.Call: property " + m + " is not a function, got " + propType.String())
		}
		panic(Error{res})
	}
	return res
	// argVals, argRefs := makeArgSlices(len(args))
	// storeArgs(args, argVals, argRefs)
	// res, ok := valueCall(v.ref, m, argRefs)
	// runtime.KeepAlive(v)
	// runtime.KeepAlive(argVals)
	// if !ok {
	// 	if vType := v.Type(); !vType.isObject() { // check here to avoid overhead in success case
	// 		panic(&ValueError{"Value.Call", vType})
	// 	}
	// 	if propType := v.Get(m).Type(); propType != TypeFunction {
	// 		panic("syscall/js: Value.Call: property " + m + " is not a function, got " + propType.String())
	// 	}
	// 	panic(Error{makeValue(res)})
	// }
	// return makeValue(res)
}

// // valueCall does a JavaScript call to the method name m of ref v with the given arguments.
// //
// // Using go:noescape is safe because no references are maintained to the
// // Go string m after the syscall returns. Additionally, the args slice
// // is only used temporarily to collect the JavaScript objects for
// // the JavaScript method invocation.
// //
// //go:wasmimport gojs syscall/js.valueCall
// //go:nosplit
// //go:noescape
// func valueCall(v ref, m string, args []ref) (ref, bool)

// Invoke does a JavaScript call of the value v with the given arguments.
// It panics if v is not a JavaScript function.
// The arguments get mapped to JavaScript values according to the ValueOf function.
func (v Value) Invoke(args ...any) (res Value) {
	var err c.Int
	if len(args) == 0 {
		res = emval_call(v, nil, 0, 0, &err)
	} else {
		vargs := make([]Value, len(args))
		for i := 0; i < len(args); i++ {
			vargs[i] = ValueOf(args[i])
		}
		res = emval_call(v, *(**Value)(unsafe.Pointer(&vargs)), c.Int(len(args)), 0, &err)
	}
	if err != 0 {
		if vType := v.Type(); vType != TypeFunction { // check here to avoid overhead in success case
			panic(&ValueError{"Value.Invoke", vType})
		}
		panic(Error{res})
	}
	return

	// argVals, argRefs := makeArgSlices(len(args))
	// storeArgs(args, argVals, argRefs)
	// res, ok := valueInvoke(v.ref, argRefs)
	// runtime.KeepAlive(v)
	// runtime.KeepAlive(argVals)
	// if !ok {
	// 	if vType := v.Type(); vType != TypeFunction { // check here to avoid overhead in success case
	// 		panic(&ValueError{"Value.Invoke", vType})
	// 	}
	// 	panic(Error{makeValue(res)})
	// }
	// return makeValue(res)
}

// valueInvoke does a JavaScript call to value v with the given arguments.
//
// Using go:noescape is safe because the args slice is only used temporarily
// to collect the JavaScript objects for the JavaScript method
// invocation.
//
// //go:wasmimport gojs syscall/js.valueInvoke
// //go:noescape
// func valueInvoke(v ref, args []ref) (ref, bool)

// New uses JavaScript's "new" operator with value v as constructor and the given arguments.
// It panics if v is not a JavaScript function.
// The arguments get mapped to JavaScript values according to the ValueOf function.
func (v Value) New(args ...any) (res Value) {
	var err c.Int
	if len(args) == 0 {
		res = emval_call(v, nil, 0, 1, &err)
	} else {
		vargs := make([]Value, len(args))
		for i := 0; i < len(args); i++ {
			vargs[i] = ValueOf(args[i])
		}
		res = emval_call(v, *(**Value)(unsafe.Pointer(&vargs)), c.Int(len(args)), 1, &err)
	}
	if err != 0 {
		if vType := v.Type(); vType != TypeFunction { // check here to avoid overhead in success case
			panic(&ValueError{"Value.Invoke", vType})
		}
		panic(Error{res})
	}
	return

	// argVals, argRefs := makeArgSlices(len(args))
	// storeArgs(args, argVals, argRefs)
	// res, ok := valueNew(v.ref, argRefs)
	// runtime.KeepAlive(v)
	// runtime.KeepAlive(argVals)
	// if !ok {
	// 	if vType := v.Type(); vType != TypeFunction { // check here to avoid overhead in success case
	// 		panic(&ValueError{"Value.Invoke", vType})
	// 	}
	// 	panic(Error{makeValue(res)})
	// }
	// return makeValue(res)
}

// valueNew uses JavaScript's "new" operator with value v as a constructor and the given arguments.
//
// Using go:noescape is safe because the args slice is only used temporarily
// to collect the JavaScript objects for the constructor execution.
//
// //go:wasmimport gojs syscall/js.valueNew
// //go:noescape
// func valueNew(v ref, args []ref) (ref, bool)

func (v Value) isNumber() bool {
	return v.ref == valueZero.ref ||
		v.ref == valueNaN.ref ||
		(v.ref != valueUndefined.ref && emval_is_number(v))
}

func (v Value) float(method string) float64 {
	if !v.isNumber() {
		panic(&ValueError{method, v.Type()})
	}
	if v.ref == valueZero.ref {
		return 0
	}
	return emval_as_double(v)
	//return *(*float64)(unsafe.Pointer(&v.ref))
}

// Float returns the value v as a float64.
// It panics if v is not a JavaScript number.
func (v Value) Float() float64 {
	return v.float("Value.Float")
}

// Int returns the value v truncated to an int.
// It panics if v is not a JavaScript number.
func (v Value) Int() int {
	return int(v.float("Value.Int"))
}

// Bool returns the value v as a bool.
// It panics if v is not a JavaScript boolean.
func (v Value) Bool() bool {
	switch v.ref {
	case valueTrue.ref:
		return true
	case valueFalse.ref:
		return false
	default:
		panic(&ValueError{"Value.Bool", v.Type()})
	}
}

// Truthy returns the JavaScript "truthiness" of the value v. In JavaScript,
// false, 0, "", null, undefined, and NaN are "falsy", and everything else is
// "truthy". See https://developer.mozilla.org/en-US/docs/Glossary/Truthy.
func (v Value) Truthy() bool {
	switch v.Type() {
	case TypeUndefined, TypeNull:
		return false
	case TypeBoolean:
		return v.Bool()
	case TypeNumber:
		return v.ref != valueNaN.ref && v.ref != valueZero.ref
	case TypeString:
		return v.String() != ""
	case TypeSymbol, TypeFunction, TypeObject:
		return true
	default:
		panic("bad type")
	}
}

// String returns the value v as a string.
// String is a special case because of Go's String method convention. Unlike the other getters,
// it does not panic if v's Type is not TypeString. Instead, it returns a string of the form "<T>"
// or "<T: V>" where T is v's type and V is a string representation of v's value.
func (v Value) String() string {
	switch v.Type() {
	case TypeString:
		return jsString(v)
	case TypeUndefined:
		return "<undefined>"
	case TypeNull:
		return "<null>"
	case TypeBoolean:
		return "<boolean: " + jsString(v) + ">"
	case TypeNumber:
		return "<number: " + jsString(v) + ">"
	case TypeSymbol:
		return "<symbol>"
	case TypeObject:
		return "<object>"
	case TypeFunction:
		return "<function>"
	default:
		panic("bad type")
	}
}

func jsString(v Value) string {
	return emval_as_string(v)
	// str, length := valuePrepareString(v.ref)
	// runtime.KeepAlive(v)
	// b := make([]byte, length)
	// valueLoadString(str, b)
	// finalizeRef(str)
	// return string(b)
}

// //go:wasmimport gojs syscall/js.valuePrepareString
// func valuePrepareString(v ref) (ref, int)

// valueLoadString loads string data located at ref v into byte slice b.
//
// Using go:noescape is safe because the byte slice is only used as a destination
// for storing the string data and references to it are not maintained.
//
// //go:wasmimport gojs syscall/js.valueLoadString
// //go:noescape
// func valueLoadString(v ref, b []byte)

// InstanceOf reports whether v is an instance of type t according to JavaScript's instanceof operator.
func (v Value) InstanceOf(t Value) bool {
	return emval_instanceof(v, t)
	// r := valueInstanceOf(v.ref, t.ref)
	// runtime.KeepAlive(v)
	// runtime.KeepAlive(t)
	// return r
}

// //go:wasmimport gojs syscall/js.valueInstanceOf
// func valueInstanceOf(v ref, t ref) bool

// A ValueError occurs when a Value method is invoked on
// a Value that does not support it. Such cases are documented
// in the description of each method.
type ValueError struct {
	Method string
	Type   Type
}

func (e *ValueError) Error() string {
	return "syscall/js: call of " + e.Method + " on " + e.Type.String()
}

// CopyBytesToGo copies bytes from src to dst.
// It panics if src is not a Uint8Array or Uint8ClampedArray.
// It returns the number of bytes copied, which will be the minimum of the lengths of src and dst.
func CopyBytesToGo(dst []byte, src Value) int {
	if !(emval_instanceof(src, uint8Array) || emval_instanceof(src, uint8ClampedArray)) {
		return 0
	}
	toCopy := src.Call("subarray", 0, len(dst))
	view := emval_memory_view_uint8(uintptr(len(dst)), *(**byte)(unsafe.Pointer(&dst)))
	view.Call("set", toCopy)
	return toCopy.Length()
	// n, ok := copyBytesToGo(dst, src.ref)
	// runtime.KeepAlive(src)
	// if !ok {
	// 	panic("syscall/js: CopyBytesToGo: expected src to be a Uint8Array or Uint8ClampedArray")
	// }
	// return n
}

/*
	// func copyBytesToGo(dst []byte, src ref) (int, bool)
	"syscall/js.copyBytesToGo": (sp) => {
		sp >>>= 0;
		const dst = loadSlice(sp + 8);
		const src = loadValue(sp + 32);
		if (!(src instanceof Uint8Array || src instanceof Uint8ClampedArray)) {
			this.mem.setUint8(sp + 48, 0);
			return;
		}
		const toCopy = src.subarray(0, dst.length);
		dst.set(toCopy);
		setInt64(sp + 40, toCopy.length);
		this.mem.setUint8(sp + 48, 1);
	},
*/

// copyBytesToGo copies bytes from src to dst.
//
// Using go:noescape is safe because the dst byte slice is only used as a dst
// copy buffer and no references to it are maintained.
//
// //go:wasmimport gojs syscall/js.copyBytesToGo
// //go:noescape
// func copyBytesToGo(dst []byte, src ref) (int, bool)

// CopyBytesToJS copies bytes from src to dst.
// It panics if dst is not a Uint8Array or Uint8ClampedArray.
// It returns the number of bytes copied, which will be the minimum of the lengths of src and dst.
func CopyBytesToJS(dst Value, src []byte) int {
	if !(emval_instanceof(dst, uint8Array) || emval_instanceof(dst, uint8ClampedArray)) {
		return 0
	}
	view := emval_memory_view_uint8(uintptr(len(src)), *(**byte)(unsafe.Pointer(&src)))
	toCopy := view.Call("subarray", 0, dst.Length())
	dst.Call("set", toCopy)
	return toCopy.Length()
	// n, ok := copyBytesToJS(dst.ref, src)
	// runtime.KeepAlive(dst)
	// if !ok {
	// 	panic("syscall/js: CopyBytesToJS: expected dst to be a Uint8Array or Uint8ClampedArray")
	// }
	// return n
}

/*
	// func copyBytesToJS(dst ref, src []byte) (int, bool)
	"syscall/js.copyBytesToJS": (sp) => {
		sp >>>= 0;
		const dst = loadValue(sp + 8);
		const src = loadSlice(sp + 16);
		if (!(dst instanceof Uint8Array || dst instanceof Uint8ClampedArray)) {
			this.mem.setUint8(sp + 48, 0);
			return;
		}
		const toCopy = src.subarray(0, dst.length);
		dst.set(toCopy);
		setInt64(sp + 40, toCopy.length);
		this.mem.setUint8(sp + 48, 1);
	},
*/

var (
	uint8Array        = emval_get_global(c.Str("Uint8Array"))
	uint8ClampedArray = emval_get_global(c.Str("Uint8ClampedArray"))
)

// copyBytesToJS copies bytes from src to dst.
//
// Using go:noescape is safe because the src byte slice is only used as a src
// copy buffer and no references to it are maintained.
//
// //go:wasmimport gojs syscall/js.copyBytesToJS
// //go:noescape
// func copyBytesToJS(dst ref, src []byte) (int, bool)
