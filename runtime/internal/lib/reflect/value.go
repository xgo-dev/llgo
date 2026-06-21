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

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflect

import (
	"errors"
	"math"
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
	"github.com/goplus/llgo/runtime/internal/clite/bitcast"
	"github.com/goplus/llgo/runtime/internal/ffi"
	"github.com/goplus/llgo/runtime/internal/runtime"
	"github.com/goplus/llgo/runtime/internal/runtime/goarch"
)

// Value is the reflection interface to a Go value.
//
// Not all methods apply to all kinds of values. Restrictions,
// if any, are noted in the documentation for each method.
// Use the Kind method to find out the kind of value before
// calling kind-specific methods. Calling a method
// inappropriate to the kind of type causes a run time panic.
//
// The zero Value represents no value.
// Its IsValid method returns false, its Kind method returns Invalid,
// its String method returns "<invalid Value>", and all other methods panic.
// Most functions and methods never return an invalid value.
// If one does, its documentation states the conditions explicitly.
//
// A Value can be used concurrently by multiple goroutines provided that
// the underlying Go value can be used concurrently for the equivalent
// direct operations.
//
// To compare two Values, compare the results of the Interface method.
// Using == on two Values does not compare the underlying values
// they represent.
type Value struct {
	// typ_ holds the type of the value represented by a Value.
	// Access using the typ method to avoid escape of v.
	typ_ *abi.Type

	// Pointer-valued data or, if flagIndir is set, pointer to data.
	// Valid when either flagIndir is set or typ.pointers() is true.
	ptr unsafe.Pointer

	// flag holds metadata about the value.
	//
	// The lowest five bits give the Kind of the value, mirroring typ.Kind().
	//
	// The next set of bits are flag bits:
	//	- flagStickyRO: obtained via unexported not embedded field, so read-only
	//	- flagEmbedRO: obtained via unexported embedded field, so read-only
	//	- flagIndir: val holds a pointer to the data
	//	- flagAddr: v.CanAddr is true (implies flagIndir and ptr is non-nil)
	//	- flagMethod: v is a method value.
	// If ifaceIndir(typ), code can assume that flagIndir is set.
	//
	// The remaining 22+ bits give a method number for method values.
	// If flag.kind() != Func, code can assume that flagMethod is unset.
	flag

	// A method value represents a curried method invocation
	// like r.Read for some receiver r. The typ+val+flag bits describe
	// the receiver r, but the flag's Kind bits say Func (methods are
	// functions), and the top bits of the flag give the method number
	// in r's type's method table.
}

type flag uintptr

const (
	flagKindWidth        = 5 // there are 27 kinds
	flagKindMask    flag = 1<<flagKindWidth - 1
	flagStickyRO    flag = 1 << 5
	flagEmbedRO     flag = 1 << 6
	flagIndir       flag = 1 << 7
	flagAddr        flag = 1 << 8
	flagMethod      flag = 1 << 9
	flagMethodShift      = 10
	flagRO          flag = flagStickyRO | flagEmbedRO
)

func (f flag) kind() Kind {
	return Kind(f & flagKindMask)
}

func (f flag) ro() flag {
	if f&flagRO != 0 {
		return flagStickyRO
	}
	return 0
}

func (v Value) typ() *abi.Type {
	// Types are either static (for compiler-created types) or
	// heap-allocated but always reachable (for reflection-created
	// types, held in the central map). So there is no need to
	// escape types. noescape here help avoid unnecessary escape
	// of v.
	return (*abi.Type)(unsafe.Pointer(v.typ_))
}

// pointer returns the underlying pointer represented by v.
// v.Kind() must be Pointer, Map, Chan, Func, or UnsafePointer
// if v.Kind() == Pointer, the base type must not be not-in-heap.
func (v Value) pointer() unsafe.Pointer {
	if v.typ_.IsClosure() {
		return v.ptr
	}
	if v.typ().Size() != goarch.PtrSize || !v.typ().Pointers() {
		panic("can't call pointer on a non-pointer Value")
	}
	if v.flag&flagIndir != 0 {
		return *(*unsafe.Pointer)(v.ptr)
	}
	return v.ptr
}

// packEface converts v to the empty interface.
func packEface(v Value) any {
	t := v.typ()
	var i any
	e := (*emptyInterface)(unsafe.Pointer(&i))
	e.word = packEfaceData(v)
	// Now, fill in the type portion. We're very careful here not
	// to have any operation between the e.word and e.typ assignments
	// that would let the garbage collector observe the partially-built
	// interface value.
	e.typ = t
	return i
}

// packEfaceData is a helper that packs the data word as if v were stored in
// an empty interface. Go 1.26's reflect.TypeAssert refers to it directly.
func packEfaceData(v Value) unsafe.Pointer {
	t := v.typ()
	switch {
	case t.IfaceIndir():
		if v.flag&flagIndir == 0 {
			panic("bad indir")
		}
		ptr := v.ptr
		if v.flag&flagAddr != 0 {
			c := unsafe_New(t)
			typedmemmove(t, c, ptr)
			ptr = c
		}
		return ptr
	case v.flag&flagIndir != 0:
		return *(*unsafe.Pointer)(v.ptr)
	default:
		return v.ptr
	}
}

// packIfaceValueIntoEmptyIface is used by Go 1.26's reflect.TypeAssert helpers
// when reboxing a non-empty interface value into an empty interface.
func packIfaceValueIntoEmptyIface(v Value) any {
	return packEface(v)
}

// unpackEface converts the empty interface i to a Value.
func unpackEface(i any) Value {
	e := (*emptyInterface)(unsafe.Pointer(&i))
	// NOTE: don't read e.word until we know whether it is really a pointer or not.
	t := e.typ
	if t == nil {
		return Value{}
	}
	f := flag(t.Kind())
	if t.IsClosure() {
		f = flag(Func)
	}
	if t.IfaceIndir() {
		f |= flagIndir
	}
	return Value{t, e.word, f}
}

// A ValueError occurs when a Value method is invoked on
// a Value that does not support it. Such cases are documented
// in the description of each method.
type ValueError struct {
	Method string
	Kind   Kind
}

func (e *ValueError) Error() string {
	if e.Kind == 0 {
		return "reflect: call of " + e.Method + " on zero Value"
	}
	return "reflect: call of " + e.Method + " on " + e.Kind.String() + " Value"
}

// valueMethodName returns the name of the exported calling method on Value.
func valueMethodName() string {
	/* TODO(xsw):
	var pc [5]uintptr
	n := runtime.Callers(1, pc[:])
	frames := runtime.CallersFrames(pc[:n])
	var frame runtime.Frame
	for more := true; more; {
		const prefix = "reflect.Value."
		frame, more = frames.Next()
		name := frame.Function
		if len(name) > len(prefix) && name[:len(prefix)] == prefix {
			methodName := name[len(prefix):]
			if len(methodName) > 0 && 'A' <= methodName[0] && methodName[0] <= 'Z' {
				return name
			}
		}
	}
	*/
	return "unknown method"
}

// emptyInterface is the header for an interface{} value.
type emptyInterface struct {
	typ  *abi.Type
	word unsafe.Pointer
}

// nonEmptyInterface is the header for an interface value with methods.
type nonEmptyInterface struct {
	// see ../runtime/iface.go:/Itab
	itab *struct {
		ityp *abi.Type // static interface type
		typ  *abi.Type // dynamic concrete type
		hash uint32    // copy of typ.hash
		_    [4]byte
		fun  [100000]unsafe.Pointer // method table
	}
	word unsafe.Pointer
}

// mustBe panics if f's kind is not expected.
// Making this a method on flag instead of on Value
// (and embedding flag in Value) means that we can write
// the very clear v.mustBe(Bool) and have it compile into
// v.flag.mustBe(Bool), which will only bother to copy the
// single important word for the receiver.
func (f flag) mustBe(expected Kind) {
	// TODO(mvdan): use f.kind() again once mid-stack inlining gets better
	if Kind(f&flagKindMask) != expected {
		panic(&ValueError{valueMethodName(), f.kind()})
	}
}

// mustBeExported panics if f records that the value was obtained using
// an unexported field.
func (f flag) mustBeExported() {
	if f == 0 || f&flagRO != 0 {
		f.mustBeExportedSlow()
	}
}

func (f flag) mustBeExportedSlow() {
	if f == 0 {
		panic(&ValueError{valueMethodName(), Invalid})
	}
	if f&flagRO != 0 {
		panic("reflect: " + valueMethodName() + " using value obtained using unexported field")
	}
}

// mustBeAssignable panics if f records that the value is not assignable,
// which is to say that either it was obtained using an unexported field
// or it is not addressable.
func (f flag) mustBeAssignable() {
	if f&flagRO != 0 || f&flagAddr == 0 {
		f.mustBeAssignableSlow()
	}
}

func (f flag) mustBeAssignableSlow() {
	if f == 0 {
		panic(&ValueError{valueMethodName(), Invalid})
	}
	// Assignable if addressable and not read-only.
	if f&flagRO != 0 {
		panic("reflect: " + valueMethodName() + " using value obtained using unexported field")
	}
	if f&flagAddr == 0 {
		panic("reflect: " + valueMethodName() + " using unaddressable value")
	}
}

// Equal reports true if v is equal to u.
// For two invalid values, Equal will report true.
// For an interface value, Equal will compare the value within the interface.
// Otherwise, If the values have different types, Equal will report false.
// Otherwise, for arrays and structs Equal will compare each element in order,
// and report false if it finds non-equal elements.
// During all comparisons, if values of the same type are compared,
// and the type is not comparable, Equal will panic.
func (v Value) Equal(u Value) bool {
	if v.Kind() == Interface {
		v = v.Elem()
	}
	if u.Kind() == Interface {
		u = u.Elem()
	}

	if !v.IsValid() || !u.IsValid() {
		return v.IsValid() == u.IsValid()
	}

	if v.Kind() != u.Kind() || v.Type() != u.Type() {
		return false
	}

	// Handle each Kind directly rather than calling valueInterface
	// to avoid allocating.
	switch v.Kind() {
	default:
		panic("reflect.Value.Equal: invalid Kind")
	case Bool:
		return v.Bool() == u.Bool()
	case Int, Int8, Int16, Int32, Int64:
		return v.Int() == u.Int()
	case Uint, Uint8, Uint16, Uint32, Uint64, Uintptr:
		return v.Uint() == u.Uint()
	case Float32, Float64:
		return v.Float() == u.Float()
	case Complex64, Complex128:
		return v.Complex() == u.Complex()
	case String:
		return v.String() == u.String()
	case Chan, Pointer, UnsafePointer:
		return v.Pointer() == u.Pointer()
	case Array:
		// u and v have the same type so they have the same length
		vl := v.Len()
		if vl == 0 {
			// panic on [0]func()
			if !v.Type().Elem().Comparable() {
				break
			}
			return true
		}
		for i := 0; i < vl; i++ {
			if !v.Index(i).Equal(u.Index(i)) {
				return false
			}
		}
		return true
	case Struct:
		// u and v have the same type so they have the same fields
		nf := v.NumField()
		for i := 0; i < nf; i++ {
			if !v.Field(i).Equal(u.Field(i)) {
				return false
			}
		}
		return true
	case Func, Map, Slice:
		break
	}
	panic("reflect.Value.Equal: values of type " + v.Type().String() + " are not comparable")
}

// Addr returns a pointer value representing the address of v.
// It panics if CanAddr() returns false.
// Addr is typically used to obtain a pointer to a struct field
// or slice element in order to call a method that requires a
// pointer receiver.
func (v Value) Addr() Value {
	if v.flag&flagAddr == 0 {
		panic("reflect.Value.Addr of unaddressable value")
	}
	// Preserve flagRO instead of using v.flag.ro() so that
	// v.Addr().Elem() is equivalent to v (#32772)
	fl := v.flag & flagRO
	return Value{ptrTo(v.typ()), v.ptr, fl | flag(Pointer)}
}

// Bool returns v's underlying value.
// It panics if v's kind is not Bool.
func (v Value) Bool() bool {
	// panicNotBool is split out to keep Bool inlineable.
	if v.kind() != Bool {
		v.panicNotBool()
	}
	if v.flag&flagIndir != 0 {
		return *(*bool)(v.ptr)
	}
	return uintptr(v.ptr) != 0
}

func (v Value) panicNotBool() {
	v.mustBe(Bool)
}

var bytesType = rtypeOf(([]byte)(nil))

// Bytes returns v's underlying value.
// It panics if v's underlying value is not a slice of bytes or
// an addressable array of bytes.
func (v Value) Bytes() []byte {
	// bytesSlow is split out to keep Bytes inlineable for unnamed []byte.
	if v.typ_ == bytesType { // ok to use v.typ_ directly as comparison doesn't cause escape
		return *(*[]byte)(v.ptr)
	}
	return v.bytesSlow()
}

func (v Value) bytesSlow() []byte {
	switch v.kind() {
	case Slice:
		if v.typ().Elem().Kind() != abi.Uint8 {
			panic("reflect.Value.Bytes of non-byte slice")
		}
		// Slice is always bigger than a word; assume flagIndir.
		return *(*[]byte)(v.ptr)
	case Array:
		if v.typ().Elem().Kind() != abi.Uint8 {
			panic("reflect.Value.Bytes of non-byte array")
		}
		if !v.CanAddr() {
			panic("reflect.Value.Bytes of unaddressable byte array")
		}
		p := (*byte)(v.ptr)
		n := int((*arrayType)(unsafe.Pointer(v.typ())).Len)
		return unsafe.Slice(p, n)
	}
	panic(&ValueError{"reflect.Value.Bytes", v.kind()})
}

// runes returns v's underlying value.
// It panics if v's underlying value is not a slice of runes (int32s).
func (v Value) runes() []rune {
	v.mustBe(Slice)
	if v.typ().Elem().Kind() != abi.Int32 {
		panic("reflect.Value.Bytes of non-rune slice")
	}
	// Slice is always bigger than a word; assume flagIndir.
	return *(*[]rune)(v.ptr)
}

// CanAddr reports whether the value's address can be obtained with Addr.
// Such values are called addressable. A value is addressable if it is
// an element of a slice, an element of an addressable array,
// a field of an addressable struct, or the result of dereferencing a pointer.
// If CanAddr returns false, calling Addr will panic.
func (v Value) CanAddr() bool {
	return v.flag&flagAddr != 0
}

// CanSet reports whether the value of v can be changed.
// A Value can be changed only if it is addressable and was not
// obtained by the use of unexported struct fields.
// If CanSet returns false, calling Set or any type-specific
// setter (e.g., SetBool, SetInt) will panic.
func (v Value) CanSet() bool {
	return v.flag&(flagAddr|flagRO) == flagAddr
}

// Cap returns v's capacity.
// It panics if v's Kind is not Array, Chan, Slice or pointer to Array.
func (v Value) Cap() int {
	// capNonSlice is split out to keep Cap inlineable for slice kinds.
	if v.kind() == Slice {
		return (*unsafeheaderSlice)(v.ptr).Cap
	}
	return v.capNonSlice()
}

func (v Value) capNonSlice() int {
	k := v.kind()
	switch k {
	case Array:
		return v.typ().Len()
	case Chan:
		return chancap(v.pointer())
	case Ptr:
		if v.typ().Elem().Kind() == abi.Array {
			return v.typ().Elem().Len()
		}
		panic("reflect: call of reflect.Value.Cap on ptr to non-array Value")
	}
	panic(&ValueError{"reflect.Value.Cap", v.kind()})
}

// Close closes the channel v.
// It panics if v's Kind is not Chan.
func (v Value) Close() {
	v.mustBe(Chan)
	v.mustBeExported()
	tt := (*chanType)(unsafe.Pointer(v.typ()))
	if ChanDir(tt.Dir)&SendDir == 0 {
		panic("reflect: close of receive-only channel")
	}
	if v.pointer() == nil {
		panic("close of nil channel")
	}
	runtime.ChanClose((*runtime.Chan)(v.pointer()))
}

// CanComplex reports whether Complex can be used without panicking.
func (v Value) CanComplex() bool {
	switch v.kind() {
	case Complex64, Complex128:
		return true
	default:
		return false
	}
}

// Complex returns v's underlying value, as a complex128.
// It panics if v's Kind is not Complex64 or Complex128
func (v Value) Complex() complex128 {
	k := v.kind()
	switch k {
	case Complex64:
		return complex128(*(*complex64)(v.ptr))
	case Complex128:
		return *(*complex128)(v.ptr)
	}
	panic(&ValueError{"reflect.Value.Complex", v.kind()})
}

// Elem returns the value that the interface v contains
// or that the pointer v points to.
// It panics if v's Kind is not Interface or Pointer.
// It returns the zero Value if v is nil.
func (v Value) Elem() Value {
	k := v.kind()
	switch k {
	case Interface:
		var eface any
		if v.typ().NumMethod() == 0 {
			eface = *(*any)(v.ptr)
		} else {
			eface = (any)(*(*interface {
				M()
			})(v.ptr))
		}
		x := unpackEface(eface)
		if x.flag != 0 {
			x.flag |= v.flag.ro()
		}
		return x
	case Pointer:
		ptr := v.ptr
		if v.flag&flagIndir != 0 {
			if ifaceIndir(v.typ()) {
				// This is a pointer to a not-in-heap object. ptr points to a uintptr
				// in the heap. That uintptr is the address of a not-in-heap object.
				// In general, pointers to not-in-heap objects can be total junk.
				// But Elem() is asking to dereference it, so the user has asserted
				// that at least it is a valid pointer (not just an integer stored in
				// a pointer slot). So let's check, to make sure that it isn't a pointer
				// that the runtime will crash on if it sees it during GC or write barriers.
				// Since it is a not-in-heap pointer, all pointers to the heap are
				// forbidden! That makes the test pretty easy.
				// See issue 48399.
				// if !verifyNotInHeapPtr(*(*uintptr)(ptr)) {
				// 	panic("reflect: reflect.Value.Elem on an invalid notinheap pointer")
				// }
			}
			ptr = *(*unsafe.Pointer)(ptr)
		}
		// The returned value's address is v's value.
		if ptr == nil {
			return Value{}
		}
		tt := (*ptrType)(unsafe.Pointer(v.typ()))
		typ := tt.Elem
		fl := v.flag&flagRO | flagIndir | flagAddr
		kind := typ.Kind()
		if typ.IsClosure() {
			kind = abi.Func
		} else if kind == abi.Func {
			typ = closureOf(typ.FuncType())
		}
		fl |= flag(kind)
		return Value{typ, ptr, fl}
	}
	panic(&ValueError{"reflect.Value.Elem", v.kind()})
}

func ifaceIndir(typ *abi.Type) bool {
	return typ.IfaceIndir()
}

// Field returns the i'th field of the struct v.
// It panics if v's Kind is not Struct or i is out of range.
func (v Value) Field(i int) Value {
	if v.kind() != Struct {
		panic(&ValueError{"reflect.Value.Field", v.kind()})
	}
	tt := (*structType)(unsafe.Pointer(v.typ()))
	if uint(i) >= uint(len(tt.Fields)) {
		panic("reflect: Field index out of range")
	}
	field := &tt.Fields[i]
	typ := field.Typ

	// Check closure to func
	kind := typ.Kind()
	if typ.IsClosure() {
		kind = abi.Func
	}
	// Inherit permission bits from v, but clear flagEmbedRO.
	fl := v.flag&(flagStickyRO|flagIndir|flagAddr) | flag(kind)
	// Using an unexported field forces flagRO.
	if !field.Exported() {
		if field.Embedded() {
			fl |= flagEmbedRO
		} else {
			fl |= flagStickyRO
		}
	}
	// Either flagIndir is set and v.ptr points at struct,
	// or flagIndir is not set and v.ptr is the actual struct data.
	// In the former case, we want v.ptr + offset.
	// In the latter case, we must have field.offset = 0,
	// so v.ptr + field.offset is still the correct address.
	ptr := add(v.ptr, field.Offset, "same as non-reflect &v.field")
	return Value{typ, ptr, fl}
}

// FieldByIndex returns the nested field corresponding to index.
// It panics if evaluation requires stepping through a nil
// pointer or a field that is not a struct.
func (v Value) FieldByIndex(index []int) Value {
	if len(index) == 1 {
		return v.Field(index[0])
	}
	v.mustBe(Struct)
	for i, x := range index {
		if i > 0 {
			if v.Kind() == Pointer && v.typ().Elem().Kind() == abi.Struct {
				if v.IsNil() {
					panic("reflect: indirection through nil pointer to embedded struct")
				}
				v = v.Elem()
			}
		}
		v = v.Field(x)
	}
	return v
}

// FieldByIndexErr returns the nested field corresponding to index.
// It returns an error if evaluation requires stepping through a nil
// pointer, but panics if it must step through a field that
// is not a struct.
func (v Value) FieldByIndexErr(index []int) (Value, error) {
	if len(index) == 1 {
		return v.Field(index[0]), nil
	}
	v.mustBe(Struct)
	for i, x := range index {
		if i > 0 {
			if v.Kind() == Ptr && v.typ().Elem().Kind() == abi.Struct {
				if v.IsNil() {
					return Value{}, errors.New("reflect: indirection through nil pointer to embedded struct field " + nameFor(v.typ().Elem()))
				}
				v = v.Elem()
			}
		}
		v = v.Field(x)
	}
	return v, nil
}

// FieldByName returns the struct field with the given name.
// It returns the zero Value if no field was found.
// It panics if v's Kind is not struct.
func (v Value) FieldByName(name string) Value {
	v.mustBe(Struct)
	if f, ok := toRType(v.typ()).FieldByName(name); ok {
		return v.FieldByIndex(f.Index)
	}
	return Value{}
}

// FieldByNameFunc returns the struct field with a name
// that satisfies the match function.
// It panics if v's Kind is not struct.
// It returns the zero Value if no field was found.
func (v Value) FieldByNameFunc(match func(string) bool) Value {
	if f, ok := toRType(v.typ()).FieldByNameFunc(match); ok {
		return v.FieldByIndex(f.Index)
	}
	return Value{}
}

// CanFloat reports whether Float can be used without panicking.
func (v Value) CanFloat() bool {
	switch v.kind() {
	case Float32, Float64:
		return true
	default:
		return false
	}
}

// Float returns v's underlying value, as a float64.
// It panics if v's Kind is not Float32 or Float64
func (v Value) Float() float64 {
	k := v.kind()
	if v.flag&flagIndir != 0 {
		switch k {
		case Float32:
			return float64(*(*float32)(v.ptr))
		case Float64:
			return *(*float64)(v.ptr)
		}
	} else {
		switch k {
		case Float32:
			return float64(bitcast.ToFloat32(int32(uintptr(v.ptr))))
		case Float64:
			if is64bit {
				return bitcast.ToFloat64(int64(uintptr(v.ptr)))
			} else {
				return *(*float64)(v.ptr)
			}
		}
	}
	panic(&ValueError{"reflect.Value.Float", v.kind()})
}

var uint8Type = rtypeOf(uint8(0))

// Index returns v's i'th element.
// It panics if v's Kind is not Array, Slice, or String or i is out of range.
func (v Value) Index(i int) Value {
	switch v.kind() {
	case Slice:
		// Element flag same as Elem of Pointer.
		// Addressable, indirect, possibly read-only.
		s := (*unsafeheaderSlice)(v.ptr)
		if uint(i) >= uint(s.Len) {
			panic("reflect: slice index out of range")
		}
		tt := (*sliceType)(unsafe.Pointer(v.typ()))
		typ := tt.Elem
		val := arrayAt(s.Data, i, typ.Size(), "i < s.Len")
		fl := flagAddr | flagIndir | v.flag.ro() | flag(typ.Kind())
		return Value{typ, val, fl}

	case String:
		s := (*unsafeheaderString)(v.ptr)
		if uint(i) >= uint(s.Len) {
			panic("reflect: string index out of range")
		}
		p := arrayAt(s.Data, i, 1, "i < s.Len")
		fl := v.flag.ro() | flag(Uint8) | flagIndir
		return Value{uint8Type, p, fl}

	case Array:
		tt := (*arrayType)(unsafe.Pointer(v.typ()))
		if uint(i) >= uint(tt.Len) {
			panic("reflect: array index out of range")
		}
		typ := tt.Elem
		offset := uintptr(i) * typ.Size()

		// Either flagIndir is set and v.ptr points at array,
		// or flagIndir is not set and v.ptr is the actual array data.
		// In the former case, we want v.ptr + offset.
		// In the latter case, we must be doing Index(0), so offset = 0,
		// so v.ptr + offset is still the correct address.
		val := add(v.ptr, offset, "same as &v[i], i < tt.len")
		fl := v.flag&(flagIndir|flagAddr) | v.flag.ro() | flag(typ.Kind()) // bits same as overall array
		return Value{typ, val, fl}
	}
	panic(&ValueError{"reflect.Value.Index", v.kind()})
}

// CanInt reports whether Int can be used without panicking.
func (v Value) CanInt() bool {
	switch v.kind() {
	case Int, Int8, Int16, Int32, Int64:
		return true
	default:
		return false
	}
}

// Int returns v's underlying value, as an int64.
// It panics if v's Kind is not Int, Int8, Int16, Int32, or Int64.
func (v Value) Int() int64 {
	f := v.flag
	k := f.kind()
	p := v.ptr
	if f&flagIndir != 0 {
		switch k {
		case Int:
			return int64(*(*int)(p))
		case Int8:
			return int64(*(*int8)(p))
		case Int16:
			return int64(*(*int16)(p))
		case Int32:
			return int64(*(*int32)(p))
		case Int64:
			return *(*int64)(p)
		}
	} else {
		switch k {
		case Int, Int8, Int16, Int32:
			return int64(uintptr(p))
		case Int64:
			if is64bit {
				return int64(uintptr(p))
			} else {
				return *(*int64)(p)
			}
		}
	}
	panic(&ValueError{"reflect.Value.Int", v.kind()})
}

// CanInterface reports whether Interface can be used without panicking.
func (v Value) CanInterface() bool {
	if v.flag == 0 {
		panic(&ValueError{"reflect.Value.CanInterface", Invalid})
	}
	return v.flag&flagRO == 0
}

// Interface returns v's current value as an interface{}.
// It is equivalent to:
//
//	var i interface{} = (v's underlying value)
//
// It panics if the Value was obtained by accessing
// unexported struct fields.
func (v Value) Interface() (i any) {
	return valueInterface(v, true)
}

func valueInterface(v Value, safe bool) any {
	if v.flag == 0 {
		panic(&ValueError{"reflect.Value.Interface", Invalid})
	}
	if safe && v.flag&flagRO != 0 {
		// Do not allow access to unexported values via Interface,
		// because they might be pointers that should not be
		// writable or methods or function that should not be callable.
		panic("reflect.Value.Interface: cannot return value obtained from unexported field or method")
	}
	if v.flag&flagMethod != 0 {
		v = makeMethodValue("Interface", v)
	}

	if v.kind() == Interface {
		// Special case: return the element inside the interface.
		// Empty interface has one layout, all interfaces with
		// methods have a second layout.
		if v.NumMethod() == 0 {
			return *(*any)(v.ptr)
		}
		return *(*interface {
			M()
		})(v.ptr)
	}

	// TODO: pass safe to packEface so we don't need to copy if safe==true?
	return packEface(v)
}

// InterfaceData returns a pair of unspecified uintptr values.
// It panics if v's Kind is not Interface.
//
// In earlier versions of Go, this function returned the interface's
// value as a uintptr pair. As of Go 1.4, the implementation of
// interface values precludes any defined use of InterfaceData.
//
// Deprecated: The memory representation of interface values is not
// compatible with InterfaceData.
func (v Value) InterfaceData() [2]uintptr {
	v.mustBe(Interface)
	// The compiler loses track as it converts to uintptr. Force escape.
	escapes(v.ptr)
	// We treat this as a read operation, so we allow
	// it even for unexported data, because the caller
	// has to import "unsafe" to turn it into something
	// that can be abused.
	// Interface value is always bigger than a word; assume flagIndir.
	return *(*[2]uintptr)(v.ptr)
}

// IsNil reports whether its argument v is nil. The argument must be
// a chan, func, interface, map, pointer, or slice value; if it is
// not, IsNil panics. Note that IsNil is not always equivalent to a
// regular comparison with nil in Go. For example, if v was created
// by calling ValueOf with an uninitialized interface variable i,
// i==nil will be true but v.IsNil will panic as v will be the zero
// Value.
func (v Value) IsNil() bool {
	k := v.kind()
	switch k {
	case Chan, Func, Map, Pointer, UnsafePointer:
		if v.flag&flagMethod != 0 {
			return false
		}
		ptr := v.ptr
		if v.flag&flagIndir != 0 {
			ptr = *(*unsafe.Pointer)(ptr)
		}
		return ptr == nil
	case Interface, Slice:
		// Both interface and slice are nil if first word is 0.
		// Both are always bigger than a word; assume flagIndir.
		return *(*unsafe.Pointer)(v.ptr) == nil
	}
	panic(&ValueError{"reflect.Value.IsNil", v.kind()})
}

// IsValid reports whether v represents a value.
// It returns false if v is the zero Value.
// If IsValid returns false, all other methods except String panic.
// Most functions and methods never return an invalid Value.
// If one does, its documentation states the conditions explicitly.
func (v Value) IsValid() bool {
	return v.flag != 0
}

// IsZero reports whether v is the zero value for its type.
// It panics if the argument is invalid.
func (v Value) IsZero() bool {
	switch v.kind() {
	case Bool:
		return !v.Bool()
	case Int, Int8, Int16, Int32, Int64:
		return v.Int() == 0
	case Uint, Uint8, Uint16, Uint32, Uint64, Uintptr:
		return v.Uint() == 0
	case Float32, Float64:
		return math.Float64bits(v.Float()) == 0
	case Complex64, Complex128:
		c := v.Complex()
		return math.Float64bits(real(c)) == 0 && math.Float64bits(imag(c)) == 0
	case Array:
		// If the type is comparable, then compare directly with zero.
		if v.typ().Equal != nil && v.typ().Size() <= maxZero {
			if v.flag&flagIndir == 0 {
				return v.ptr == nil
			}
			// v.ptr doesn't escape, as Equal functions are compiler generated
			// and never escape. The escape analysis doesn't know, as it is a
			// function pointer call.
			return v.typ().Equal(noescape(v.ptr), unsafe.Pointer(&zeroVal[0]))
		}

		n := v.Len()
		for i := 0; i < n; i++ {
			if !v.Index(i).IsZero() {
				return false
			}
		}
		return true
	case Chan, Func, Interface, Map, Pointer, Slice, UnsafePointer:
		return v.IsNil()
	case String:
		return v.Len() == 0
	case Struct:
		// If the type is comparable, then compare directly with zero.
		if v.typ().Equal != nil && v.typ().Size() <= maxZero {
			if v.flag&flagIndir == 0 {
				return v.ptr == nil
			}
			// See noescape justification above.
			return v.typ().Equal(noescape(v.ptr), unsafe.Pointer(&zeroVal[0]))
		}

		n := v.NumField()
		for i := 0; i < n; i++ {
			if !v.Field(i).IsZero() {
				return false
			}
		}
		return true
	default:
		// This should never happen, but will act as a safeguard for later,
		// as a default value doesn't makes sense here.
		panic(&ValueError{"reflect.Value.IsZero", v.Kind()})
	}
}

// SetZero sets v to be the zero value of v's type.
// It panics if CanSet returns false.
func (v Value) SetZero() {
	v.mustBeAssignable()
	switch v.kind() {
	case Bool:
		*(*bool)(v.ptr) = false
	case Int:
		*(*int)(v.ptr) = 0
	case Int8:
		*(*int8)(v.ptr) = 0
	case Int16:
		*(*int16)(v.ptr) = 0
	case Int32:
		*(*int32)(v.ptr) = 0
	case Int64:
		*(*int64)(v.ptr) = 0
	case Uint:
		*(*uint)(v.ptr) = 0
	case Uint8:
		*(*uint8)(v.ptr) = 0
	case Uint16:
		*(*uint16)(v.ptr) = 0
	case Uint32:
		*(*uint32)(v.ptr) = 0
	case Uint64:
		*(*uint64)(v.ptr) = 0
	case Uintptr:
		*(*uintptr)(v.ptr) = 0
	case Float32:
		*(*float32)(v.ptr) = 0
	case Float64:
		*(*float64)(v.ptr) = 0
	case Complex64:
		*(*complex64)(v.ptr) = 0
	case Complex128:
		*(*complex128)(v.ptr) = 0
	case String:
		*(*string)(v.ptr) = ""
	case Slice:
		*(*unsafeheaderSlice)(v.ptr) = unsafeheaderSlice{}
	case Interface:
		*(*[2]unsafe.Pointer)(v.ptr) = [2]unsafe.Pointer{}
	case Chan, Func, Map, Pointer, UnsafePointer:
		*(*unsafe.Pointer)(v.ptr) = nil
	case Array, Struct:
		typedmemclr(v.typ(), v.ptr)
	default:
		// This should never happen, but will act as a safeguard for later,
		// as a default value doesn't makes sense here.
		panic(&ValueError{"reflect.Value.SetZero", v.Kind()})
	}
}

// Kind returns v's Kind.
// If v is the zero Value (IsValid returns false), Kind returns Invalid.
func (v Value) Kind() Kind {
	return v.kind()
}

// Len returns v's length.
// It panics if v's Kind is not Array, Chan, Map, Slice, String, or pointer to Array.
func (v Value) Len() int {
	// lenNonSlice is split out to keep Len inlineable for slice kinds.
	if v.kind() == Slice {
		return (*unsafeheaderSlice)(v.ptr).Len
	}
	return v.lenNonSlice()
}

func (v Value) lenNonSlice() int {
	switch k := v.kind(); k {
	case Array:
		tt := (*arrayType)(unsafe.Pointer(v.typ()))
		return int(tt.Len)
	case Chan:
		return chanlen(v.pointer())
	case Map:
		return maplen(v.pointer())
	case String:
		// String is bigger than a word; assume flagIndir.
		return (*unsafeheaderString)(v.ptr).Len
	case Ptr:
		if v.typ().Elem().Kind() == abi.Array {
			return v.typ().Elem().Len()
		}
		panic("reflect: call of reflect.Value.Len on ptr to non-array Value")
	}
	panic(&ValueError{"reflect.Value.Len", v.kind()})
}

// Pointer returns v's value as a uintptr.
// It panics if v's Kind is not Chan, Func, Map, Pointer, Slice, or UnsafePointer.
//
// If v's Kind is Func, the returned pointer is an underlying
// code pointer, but not necessarily enough to identify a
// single function uniquely. The only guarantee is that the
// result is zero if and only if v is a nil func Value.
//
// If v's Kind is Slice, the returned pointer is to the first
// element of the slice. If the slice is nil the returned value
// is 0.  If the slice is empty but non-nil the return value is non-zero.
//
// It's preferred to use uintptr(Value.UnsafePointer()) to get the equivalent result.
func (v Value) Pointer() uintptr {
	// The compiler loses track as it converts to uintptr. Force escape.
	escapes(v.ptr)

	k := v.kind()
	switch k {
	case Pointer:
		if v.typ().PtrBytes == 0 {
			val := *(*uintptr)(v.ptr)
			// Since it is a not-in-heap pointer, all pointers to the heap are
			// forbidden! See comment in Value.Elem and issue #48399.
			// if !verifyNotInHeapPtr(val) {
			// 	panic("reflect: reflect.Value.Pointer on an invalid notinheap pointer")
			// }
			return val
		}
		fallthrough
	case Chan, Map, UnsafePointer:
		return uintptr(v.pointer())
	case Func:
		if v.flag&flagMethod != 0 {
			// As the doc comment says, the returned pointer is an
			// underlying code pointer but not necessarily enough to
			// identify a single function uniquely. All method expressions
			// created via reflect have the same underlying code pointer,
			// so their Pointers are equal. The function used here must
			// match the one used in makeMethodValue.
			//			return methodValueCallCodePtr()
			_, _, fn := methodReceiver("unsafePointer", v, int(v.flag)>>flagMethodShift)
			return uintptr(fn)
		}
		p := v.pointer()
		// Non-nil func value points at data block.
		// First word of data block is actual code.
		if p != nil && v.typ_.IsClosure() {
			p = *(*unsafe.Pointer)(p)
		}
		return uintptr(p)

	case Slice:
		return uintptr((*unsafeheaderSlice)(v.ptr).Data)
	}
	panic(&ValueError{"reflect.Value.Pointer", v.kind()})
}

// Recv receives and returns a value from the channel v.
// It panics if v's Kind is not Chan.
// The receive blocks until a value is ready.
// The boolean value ok is true if the value x corresponds to a send
// on the channel, false if it is a zero value received because the channel is closed.
func (v Value) Recv() (x Value, ok bool) {
	v.mustBe(Chan)
	v.mustBeExported()
	return v.recv(false)
}

// internal recv, possibly non-blocking (nb).
// v is known to be a channel.
func (v Value) recv(nb bool) (val Value, ok bool) {
	tt := (*chanType)(unsafe.Pointer(v.typ()))
	if ChanDir(tt.Dir)&RecvDir == 0 {
		panic("reflect: recv on send-only channel")
	}
	t := tt.Elem
	val = Value{t, nil, flag(t.Kind())}
	var p unsafe.Pointer
	if ifaceIndir(t) {
		p = unsafe_New(t)
		val.ptr = p
		val.flag |= flagIndir
	} else {
		p = unsafe.Pointer(&val.ptr)
	}
	ch := (*runtime.Chan)(v.pointer())
	if ch == nil {
		if nb {
			return Value{}, false
		}
		select {}
	}
	selected := true
	if nb {
		var tryOK bool
		ok, tryOK = runtime.ChanTryRecv(ch, p, int(t.Size()))
		selected = tryOK
	} else {
		ok = runtime.ChanRecv(ch, p, int(t.Size()))
	}
	if !selected {
		val = Value{}
	}
	return
}

// Send sends x on the channel v.
// It panics if v's kind is not Chan or if x's type is not the same type as v's element type.
// As in Go, x's value must be assignable to the channel's element type.
func (v Value) Send(x Value) {
	v.mustBe(Chan)
	v.mustBeExported()
	v.send(x, false)
}

// internal send, possibly non-blocking.
// v is known to be a channel.
func (v Value) send(x Value, nb bool) (selected bool) {
	tt := (*chanType)(unsafe.Pointer(v.typ()))
	if ChanDir(tt.Dir)&SendDir == 0 {
		panic("reflect: send on recv-only channel")
	}
	x.mustBeExported()
	x = x.assignTo("reflect.Value.Send", tt.Elem, nil)
	var p unsafe.Pointer
	if x.flag&flagIndir != 0 {
		p = x.ptr
	} else {
		p = unsafe.Pointer(&x.ptr)
	}
	ch := (*runtime.Chan)(v.pointer())
	if ch == nil {
		if nb {
			return false
		}
		select {}
	}
	if nb {
		return runtime.ChanTrySend(ch, p, int(tt.Elem.Size()))
	}
	if !runtime.ChanSend(ch, p, int(tt.Elem.Size())) {
		panic("send on closed channel")
	}
	return true
}

// Set assigns x to the value v.
// It panics if CanSet returns false.
// As in Go, x's value must be assignable to v's type and
// must not be derived from an unexported field.
func (v Value) Set(x Value) {
	v.mustBeAssignable()
	x.mustBeExported() // do not let unexported x leak
	var target unsafe.Pointer
	if v.kind() == Interface {
		target = v.ptr
	}
	x = x.assignTo("reflect.Set", v.typ(), target)
	if x.flag&flagIndir != 0 {
		if x.ptr == unsafe.Pointer(&runtime.ZeroVal[0]) {
			typedmemclr(v.typ(), v.ptr)
		} else {
			typedmemmove(v.typ(), v.ptr, x.ptr)
		}
	} else {
		*(*unsafe.Pointer)(v.ptr) = x.ptr
	}
}

// SetBool sets v's underlying value.
// It panics if v's Kind is not Bool or if CanSet() is false.
func (v Value) SetBool(x bool) {
	v.mustBeAssignable()
	v.mustBe(Bool)
	*(*bool)(v.ptr) = x
}

// SetBytes sets v's underlying value.
// It panics if v's underlying value is not a slice of bytes.
func (v Value) SetBytes(x []byte) {
	v.mustBeAssignable()
	v.mustBe(Slice)
	if toRType(v.typ()).Elem().Kind() != Uint8 { // TODO add Elem method, fix mustBe(Slice) to return slice.
		panic("reflect.Value.SetBytes of non-byte slice")
	}
	*(*[]byte)(v.ptr) = x
}

// setRunes sets v's underlying value.
// It panics if v's underlying value is not a slice of runes (int32s).
func (v Value) setRunes(x []rune) {
	v.mustBeAssignable()
	v.mustBe(Slice)
	if v.typ().Elem().Kind() != abi.Int32 {
		panic("reflect.Value.setRunes of non-rune slice")
	}
	*(*[]rune)(v.ptr) = x
}

// SetComplex sets v's underlying value to x.
// It panics if v's Kind is not Complex64 or Complex128, or if CanSet() is false.
func (v Value) SetComplex(x complex128) {
	v.mustBeAssignable()
	switch k := v.kind(); k {
	default:
		panic(&ValueError{"reflect.Value.SetComplex", v.kind()})
	case Complex64:
		*(*complex64)(v.ptr) = complex64(x)
	case Complex128:
		*(*complex128)(v.ptr) = x
	}
}

// SetFloat sets v's underlying value to x.
// It panics if v's Kind is not Float32 or Float64, or if CanSet() is false.
func (v Value) SetFloat(x float64) {
	v.mustBeAssignable()
	switch k := v.kind(); k {
	default:
		panic(&ValueError{"reflect.Value.SetFloat", v.kind()})
	case Float32:
		*(*float32)(v.ptr) = float32(x)
	case Float64:
		*(*float64)(v.ptr) = x
	}
}

// SetInt sets v's underlying value to x.
// It panics if v's Kind is not Int, Int8, Int16, Int32, or Int64, or if CanSet() is false.
func (v Value) SetInt(x int64) {
	v.mustBeAssignable()
	switch k := v.kind(); k {
	default:
		panic(&ValueError{"reflect.Value.SetInt", v.kind()})
	case Int:
		*(*int)(v.ptr) = int(x)
	case Int8:
		*(*int8)(v.ptr) = int8(x)
	case Int16:
		*(*int16)(v.ptr) = int16(x)
	case Int32:
		*(*int32)(v.ptr) = int32(x)
	case Int64:
		*(*int64)(v.ptr) = x
	}
}

// SetLen sets v's length to n.
// It panics if v's Kind is not Slice or if n is negative or
// greater than the capacity of the slice.
func (v Value) SetLen(n int) {
	v.mustBeAssignable()
	v.mustBe(Slice)
	s := (*unsafeheaderSlice)(v.ptr)
	if uint(n) > uint(s.Cap) {
		panic("reflect: slice length out of range in SetLen")
	}
	s.Len = n
}

// SetCap sets v's capacity to n.
// It panics if v's Kind is not Slice or if n is smaller than the length or
// greater than the capacity of the slice.
func (v Value) SetCap(n int) {
	v.mustBeAssignable()
	v.mustBe(Slice)
	s := (*unsafeheaderSlice)(v.ptr)
	if n < s.Len || n > s.Cap {
		panic("reflect: slice capacity out of range in SetCap")
	}
	s.Cap = n
}

// SetMapIndex sets the element associated with key in the map v to elem.
// It panics if v's Kind is not Map.
// If elem is the zero Value, SetMapIndex deletes the key from the map.
// Otherwise if v holds a nil map, SetMapIndex will panic.
// As in Go, key's elem must be assignable to the map's key type,
// and elem's value must be assignable to the map's elem type.
func (v Value) SetMapIndex(key, elem Value) {
	v.mustBe(Map)
	v.mustBeExported()
	key.mustBeExported()
	tt := (*mapType)(unsafe.Pointer(v.typ()))

	// if (tt.Key == stringType || key.kind() == String) && tt.Key == key.typ() && tt.Elem.Size() <= maxValSize {
	// 	k := *(*string)(key.ptr)
	// 	if elem.typ() == nil {
	// 		mapdelete_faststr(v.typ(), v.pointer(), k)
	// 		return
	// 	}
	// 	elem.mustBeExported()
	// 	elem = elem.assignTo("reflect.Value.SetMapIndex", tt.Elem, nil)
	// 	var e unsafe.Pointer
	// 	if elem.flag&flagIndir != 0 {
	// 		e = elem.ptr
	// 	} else {
	// 		e = unsafe.Pointer(&elem.ptr)
	// 	}
	// 	mapassign_faststr(v.typ(), v.pointer(), k, e)
	// 	return
	// }

	key = key.assignTo("reflect.Value.SetMapIndex", tt.Key, nil)
	var k unsafe.Pointer
	if key.flag&flagIndir != 0 {
		k = key.ptr
	} else {
		k = unsafe.Pointer(&key.ptr)
	}
	if elem.typ() == nil {
		mapdelete(v.typ(), v.pointer(), k)
		return
	}
	elem.mustBeExported()
	elem = elem.assignTo("reflect.Value.SetMapIndex", tt.Elem, nil)
	var e unsafe.Pointer
	if elem.flag&flagIndir != 0 {
		e = elem.ptr
	} else {
		e = unsafe.Pointer(&elem.ptr)
	}
	mapassign(v.typ(), v.pointer(), k, e)
}

// SetUint sets v's underlying value to x.
// It panics if v's Kind is not Uint, Uintptr, Uint8, Uint16, Uint32, or Uint64, or if CanSet() is false.
func (v Value) SetUint(x uint64) {
	v.mustBeAssignable()
	switch k := v.kind(); k {
	default:
		panic(&ValueError{"reflect.Value.SetUint", v.kind()})
	case Uint:
		*(*uint)(v.ptr) = uint(x)
	case Uint8:
		*(*uint8)(v.ptr) = uint8(x)
	case Uint16:
		*(*uint16)(v.ptr) = uint16(x)
	case Uint32:
		*(*uint32)(v.ptr) = uint32(x)
	case Uint64:
		*(*uint64)(v.ptr) = x
	case Uintptr:
		*(*uintptr)(v.ptr) = uintptr(x)
	}
}

// SetPointer sets the [unsafe.Pointer] value v to x.
// It panics if v's Kind is not UnsafePointer.
func (v Value) SetPointer(x unsafe.Pointer) {
	v.mustBeAssignable()
	v.mustBe(UnsafePointer)
	*(*unsafe.Pointer)(v.ptr) = x
}

// SetString sets v's underlying value to x.
// It panics if v's Kind is not String or if CanSet() is false.
func (v Value) SetString(x string) {
	v.mustBeAssignable()
	v.mustBe(String)
	*(*string)(v.ptr) = x
}

// Slice returns v[i:j].
// It panics if v's Kind is not Array, Slice or String, or if v is an unaddressable array,
// or if the indexes are out of bounds.
func (v Value) Slice(i, j int) Value {
	var (
		cap  int
		typ  *sliceType
		base unsafe.Pointer
	)
	switch kind := v.kind(); kind {
	default:
		panic(&ValueError{"reflect.Value.Slice", v.kind()})

	case Array:
		if v.flag&flagAddr == 0 {
			panic("reflect.Value.Slice: slice of unaddressable array")
		}
		tt := (*arrayType)(unsafe.Pointer(v.typ()))
		cap = int(tt.Len)
		typ = (*sliceType)(unsafe.Pointer(tt.Slice))
		base = v.ptr

	case Slice:
		typ = (*sliceType)(unsafe.Pointer(v.typ()))
		s := (*unsafeheaderSlice)(v.ptr)
		base = s.Data
		cap = s.Cap

	case String:
		s := (*unsafeheaderString)(v.ptr)
		if i < 0 || j < i || j > s.Len {
			panic("reflect.Value.Slice: string slice index out of bounds")
		}
		var t unsafeheaderString
		if i < s.Len {
			t = unsafeheaderString{Data: arrayAt(s.Data, i, 1, "i < s.Len"), Len: j - i}
		}
		return Value{v.typ(), unsafe.Pointer(&t), v.flag}
	}

	if i < 0 || j < i || j > cap {
		panic("reflect.Value.Slice: slice index out of bounds")
	}

	// Declare slice so that gc can see the base pointer in it.
	var x []unsafe.Pointer

	// Reinterpret as *unsafeheader.Slice to edit.
	s := (*unsafeheaderSlice)(unsafe.Pointer(&x))
	s.Len = j - i
	s.Cap = cap - i
	if cap-i > 0 {
		s.Data = arrayAt(base, i, typ.Elem.Size(), "i < cap")
	} else {
		// do not advance pointer, to avoid pointing beyond end of slice
		s.Data = base
	}

	fl := v.flag.ro() | flagIndir | flag(Slice)
	return Value{typ.Common(), unsafe.Pointer(&x), fl}
}

// Slice3 is the 3-index form of the slice operation: it returns v[i:j:k].
// It panics if v's Kind is not Array or Slice, or if v is an unaddressable array,
// or if the indexes are out of bounds.
func (v Value) Slice3(i, j, k int) Value {
	var (
		cap  int
		typ  *sliceType
		base unsafe.Pointer
	)
	switch kind := v.kind(); kind {
	default:
		panic(&ValueError{"reflect.Value.Slice3", v.kind()})

	case Array:
		if v.flag&flagAddr == 0 {
			panic("reflect.Value.Slice3: slice of unaddressable array")
		}
		tt := (*arrayType)(unsafe.Pointer(v.typ()))
		cap = int(tt.Len)
		typ = (*sliceType)(unsafe.Pointer(tt.Slice))
		base = v.ptr

	case Slice:
		typ = (*sliceType)(unsafe.Pointer(v.typ()))
		s := (*unsafeheaderSlice)(v.ptr)
		base = s.Data
		cap = s.Cap
	}

	if i < 0 || j < i || k < j || k > cap {
		panic("reflect.Value.Slice3: slice index out of bounds")
	}

	// Declare slice so that the garbage collector
	// can see the base pointer in it.
	var x []unsafe.Pointer

	// Reinterpret as *unsafeheader.Slice to edit.
	s := (*unsafeheaderSlice)(unsafe.Pointer(&x))
	s.Len = j - i
	s.Cap = k - i
	if k-i > 0 {
		s.Data = arrayAt(base, i, typ.Elem.Size(), "i < k <= cap")
	} else {
		// do not advance pointer, to avoid pointing beyond end of slice
		s.Data = base
	}

	fl := v.flag.ro() | flagIndir | flag(Slice)
	return Value{typ.Common(), unsafe.Pointer(&x), fl}
}

// String returns the string v's underlying value, as a string.
// String is a special case because of Go's String method convention.
// Unlike the other getters, it does not panic if v's Kind is not String.
// Instead, it returns a string of the form "<T value>" where T is v's type.
// The fmt package treats Values specially. It does not call their String
// method implicitly but instead prints the concrete values they hold.
func (v Value) String() string {
	// stringNonString is split out to keep String inlineable for string kinds.
	if v.kind() == String {
		return *(*string)(v.ptr)
	}
	return v.stringNonString()
}

func (v Value) stringNonString() string {
	if v.kind() == Invalid {
		return "<invalid Value>"
	}
	// If you call String on a reflect.Value of other type, it's better to
	// print something than to panic. Useful in debugging.
	return "<" + v.Type().String() + " Value>"
}

// TryRecv attempts to receive a value from the channel v but will not block.
// It panics if v's Kind is not Chan.
// If the receive delivers a value, x is the transferred value and ok is true.
// If the receive cannot finish without blocking, x is the zero Value and ok is false.
// If the channel is closed, x is the zero value for the channel's element type and ok is false.
func (v Value) TryRecv() (x Value, ok bool) {
	v.mustBe(Chan)
	v.mustBeExported()
	return v.recv(true)
}

// TrySend attempts to send x on the channel v but will not block.
// It panics if v's Kind is not Chan.
// It reports whether the value was sent.
// As in Go, x's value must be assignable to the channel's element type.
func (v Value) TrySend(x Value) bool {
	v.mustBe(Chan)
	v.mustBeExported()
	return v.send(x, true)
}

// Type returns v's type.
func (v Value) Type() Type {
	if v.flag != 0 && v.flag&flagMethod == 0 && !v.typ_.IsClosure() {
		return (*rtype)(unsafe.Pointer(v.typ_)) // inline of toRType(v.typ()), for own inlining in inline test
	}
	return v.typeSlow()
}

func (v Value) typeSlow() Type {
	if v.flag == 0 {
		panic(&ValueError{"reflect.Value.Type", Invalid})
	}

	typ := v.typ()
	if v.flag&flagMethod == 0 {
		// closure func
		if v.typ_.IsClosure() {
			return toRType(&v.closureFunc().Type)
		}
		return toRType(v.typ())
	}

	// Method value.
	// v.typ describes the receiver, not the method type.
	i := int(v.flag) >> flagMethodShift
	if v.typ().Kind() == abi.Interface {
		// Method on interface.
		tt := (*interfaceType)(unsafe.Pointer(typ))
		if uint(i) >= uint(len(tt.Methods)) {
			panic("reflect: internal error: invalid method index")
		}
		m := &tt.Methods[i]
		return toRType(&m.Typ_.Type)
	}
	// Method on concrete type.
	ms := typ.ExportedMethods()
	if uint(i) >= uint(len(ms)) {
		panic("reflect: internal error: invalid method index")
	}
	m := ms[i]
	return toRType(&m.Mtyp_.Type)
}

// CanUint reports whether Uint can be used without panicking.
func (v Value) CanUint() bool {
	switch v.kind() {
	case Uint, Uint8, Uint16, Uint32, Uint64, Uintptr:
		return true
	default:
		return false
	}
}

// Uint returns v's underlying value, as a uint64.
// It panics if v's Kind is not Uint, Uintptr, Uint8, Uint16, Uint32, or Uint64.
func (v Value) Uint() uint64 {
	f := v.flag
	k := v.kind()
	p := v.ptr
	if f&flagIndir != 0 {
		switch k {
		case Uint:
			return uint64(*(*uint)(p))
		case Uint8:
			return uint64(*(*uint8)(p))
		case Uint16:
			return uint64(*(*uint16)(p))
		case Uint32:
			return uint64(*(*uint32)(p))
		case Uint64:
			return *(*uint64)(p)
		case Uintptr:
			return uint64(*(*uintptr)(p))
		}
	} else {
		switch k {
		case Uint, Uint8, Uint16, Uint32:
			return uint64(uintptr(p))
		case Uint64, Uintptr:
			if is64bit {
				return uint64(uintptr(p))
			} else {
				return *(*uint64)(p)
			}
		}
	}
	panic(&ValueError{"reflect.Value.Uint", v.kind()})
}

// This prevents inlining Value.UnsafeAddr when -d=checkptr is enabled,
// which ensures cmd/compile can recognize unsafe.Pointer(v.UnsafeAddr())
// and make an exception.

// UnsafeAddr returns a pointer to v's data, as a uintptr.
// It panics if v is not addressable.
//
// It's preferred to use uintptr(Value.Addr().UnsafePointer()) to get the equivalent result.
func (v Value) UnsafeAddr() uintptr {
	if v.typ() == nil {
		panic(&ValueError{"reflect.Value.UnsafeAddr", Invalid})
	}
	if v.flag&flagAddr == 0 {
		panic("reflect.Value.UnsafeAddr of unaddressable value")
	}
	return uintptr(v.ptr)
}

// UnsafePointer returns v's value as a [unsafe.Pointer].
// It panics if v's Kind is not Chan, Func, Map, Pointer, Slice, or UnsafePointer.
//
// If v's Kind is Func, the returned pointer is an underlying
// code pointer, but not necessarily enough to identify a
// single function uniquely. The only guarantee is that the
// result is zero if and only if v is a nil func Value.
//
// If v's Kind is Slice, the returned pointer is to the first
// element of the slice. If the slice is nil the returned value
// is nil.  If the slice is empty but non-nil the return value is non-nil.
func (v Value) UnsafePointer() unsafe.Pointer {
	k := v.kind()
	switch k {
	case Pointer:
		if v.typ().PtrBytes == 0 {
			// Since it is a not-in-heap pointer, all pointers to the heap are
			// forbidden! See comment in Value.Elem and issue #48399.
			if !verifyNotInHeapPtr(*(*uintptr)(v.ptr)) {
				panic("reflect: reflect.Value.UnsafePointer on an invalid notinheap pointer")
			}
			return *(*unsafe.Pointer)(v.ptr)
		}
		fallthrough
	case Chan, Map, UnsafePointer:
		return v.pointer()
	case Func:
		if v.flag&flagMethod != 0 {
			// As the doc comment says, the returned pointer is an
			// underlying code pointer but not necessarily enough to
			// identify a single function uniquely. All method expressions
			// created via reflect have the same underlying code pointer,
			// so their Pointers are equal. The function used here must
			// match the one used in makeMethodValue.
			_, _, fn := methodReceiver("unsafePointer", v, int(v.flag)>>flagMethodShift)
			return fn
		}
		p := v.pointer()
		// Non-nil func value points at data block.
		// First word of data block is actual code.
		if p != nil && v.typ_.IsClosure() {
			p = *(*unsafe.Pointer)(p)
		}
		return p

	case Slice:
		return (*unsafeheaderSlice)(v.ptr).Data
	}
	panic(&ValueError{"reflect.Value.UnsafePointer", v.kind()})
}

//go:linkname unsafe_New github.com/goplus/llgo/runtime/internal/runtime.New
func unsafe_New(*abi.Type) unsafe.Pointer

//go:linkname unsafe_NewArray github.com/goplus/llgo/runtime/internal/runtime.NewArray
func unsafe_NewArray(*abi.Type, int) unsafe.Pointer

// ValueOf returns a new Value initialized to the concrete value
// stored in the interface i. ValueOf(nil) returns the zero Value.
func ValueOf(i any) Value {
	if i == nil {
		return Value{}
	}

	return unpackEface(i)
}

// Indirect returns the value that v points to.
// If v is a nil pointer, Indirect returns a zero Value.
// If v is not a pointer, Indirect returns v.
func Indirect(v Value) Value {
	if v.Kind() != Pointer {
		return v
	}
	return v.Elem()
}

func typesMustMatch(what string, t1, t2 Type) {
	if t1 != t2 {
		panic(what + ": " + t1.String() + " != " + t2.String())
	}
}

// arrayAt returns the i-th element of p,
// an array whose elements are eltSize bytes wide.
// The array pointed at by p must have at least i+1 elements:
// it is invalid (but impossible to check here) to pass i >= len,
// because then the result will point outside the array.
// whySafe must explain why i < len. (Passing "i < len" is fine;
// the benefit is to surface this assumption at the call site.)
func arrayAt(p unsafe.Pointer, i int, eltSize uintptr, whySafe string) unsafe.Pointer {
	return add(p, uintptr(i)*eltSize, "i < len")
}

// Grow increases the slice's capacity, if necessary, to guarantee space for
// another n elements. After Grow(n), at least n elements can be appended
// to the slice without another allocation.
//
// It panics if v's Kind is not a Slice or if n is negative or too large to
// allocate the memory.
func (v Value) Grow(n int) {
	v.mustBeAssignable()
	v.mustBe(Slice)
	v.grow(n)
}

// grow is identical to Grow but does not check for assignability.
func (v Value) grow(n int) {
	p := (*unsafeheaderSlice)(v.ptr)
	oldLen := p.Len
	switch {
	case n < 0:
		panic("reflect.Value.Grow: negative len")
	case oldLen+n < 0:
		panic("reflect.Value.Grow: slice overflow")
	case oldLen+n > p.Cap:
		t := v.typ().Elem()
		// The linknamed growslice must use the same ABI as runtime slice helpers.
		// LLGo lowers slice-like values (data,len,cap) as 3 registers, while a
		// plain 3-word struct may use a different calling convention.
		// Reinterpret the header as a builtin slice so the call ABI matches.
		sh := *(*[]byte)(unsafe.Pointer(p))
		sh = growslice(sh, n, int(t.Size_))
		*p = *(*unsafeheaderSlice)(unsafe.Pointer(&sh))
		p.Len = oldLen // set oldLen back
	}
}

// extendSlice extends a slice by n elements.
//
// Unlike Value.grow, which modifies the slice in place and
// does not change the length of the slice in place,
// extendSlice returns a new slice value with the length
// incremented by the number of specified elements.
func (v Value) extendSlice(n int) Value {
	v.mustBeExported()
	v.mustBe(Slice)

	// Shallow copy the slice header to avoid mutating the source slice.
	sh := *(*unsafeheaderSlice)(v.ptr)
	s := &sh
	v.ptr = unsafe.Pointer(s)
	v.flag = flagIndir | flag(Slice) // equivalent flag to MakeSlice

	v.grow(n) // fine to treat as assignable since we allocate a new slice header
	s.Len += n
	return v
}

// Append appends the values x to a slice s and returns the resulting slice.
// As in Go, each x's value must be assignable to the slice's element type.
func Append(s Value, x ...Value) Value {
	s.mustBe(Slice)
	n := s.Len()
	s = s.extendSlice(len(x))
	for i, v := range x {
		s.Index(n + i).Set(v)
	}
	return s
}

// AppendSlice appends a slice t to a slice s and returns the resulting slice.
// The slices s and t must have the same element type.
func AppendSlice(s, t Value) Value {
	s.mustBe(Slice)
	t.mustBe(Slice)
	if s.typ().Elem() != t.typ().Elem() {
		panic("reflect.AppendSlice: " + stringFor(s.typ().Elem()) + " != " + stringFor(t.typ().Elem()))
	}
	ns := s.Len()
	nt := t.Len()
	s = s.extendSlice(nt)
	Copy(s.Slice(ns, ns+nt), t)
	return s
}

// Copy copies the contents of src into dst until either
// dst has been filled or src has been exhausted.
// It returns the number of elements copied.
// Dst and src each must have kind Array or Slice, and
// dst and src must have the same element type.
// If dst is an Array, it panics if Value.CanSet returns false.
//
// As a special case, src can have kind String if the element type of dst is kind Uint8.
func Copy(dst, src Value) int {
	dk := dst.kind()
	if dk != Array && dk != Slice {
		panic(&ValueError{"reflect.Copy", dk})
	}
	if dk == Array {
		dst.mustBeAssignable()
	}
	dst.mustBeExported()

	sk := src.kind()
	stringCopy := false
	if sk != Array && sk != Slice {
		stringCopy = sk == String && dst.typ().Elem().Kind() == abi.Uint8
		if !stringCopy {
			panic(&ValueError{"reflect.Copy", sk})
		}
	}
	src.mustBeExported()

	de := dst.typ().Elem()
	if !stringCopy {
		se := src.typ().Elem()
		typesMustMatch("reflect.Copy", toType(de), toType(se))
	}

	var ds, ss unsafeheaderSlice
	if dk == Array {
		ds.Data = dst.ptr
		ds.Len = dst.Len()
		ds.Cap = ds.Len
	} else {
		ds = *(*unsafeheaderSlice)(dst.ptr)
	}
	if sk == Array {
		ss.Data = src.ptr
		ss.Len = src.Len()
		ss.Cap = ss.Len
	} else if sk == Slice {
		ss = *(*unsafeheaderSlice)(src.ptr)
	} else {
		sh := *(*unsafeheaderString)(src.ptr)
		ss.Data = sh.Data
		ss.Len = sh.Len
		ss.Cap = sh.Len
	}

	return typedslicecopy(de, ds, ss)
}

// Zero returns a Value representing the zero value for the specified type.
// The result is different from the zero value of the Value struct,
// which represents no value at all.
// For example, Zero(TypeOf(42)) returns a Value with Kind Int and value 0.
// The returned value is neither addressable nor settable.
func Zero(typ Type) Value {
	if typ == nil {
		panic("reflect: Zero(nil)")
	}
	t := &typ.(*rtype).t
	fl := flag(t.Kind())
	if t.IfaceIndir() {
		var p unsafe.Pointer
		if t.Size() <= maxZero {
			p = unsafe.Pointer(&runtime.ZeroVal[0])
		} else {
			p = unsafe_New(t)
		}
		return Value{t, p, fl | flagIndir}
	}
	return Value{t, nil, fl}
}

// TODO(xsw): check this
// must match declarations in runtime/map.go.
const maxZero = runtime.MaxZero

//go:linkname zeroVal runtime.ZeroVal
var zeroVal [maxZero]byte

// New returns a Value representing a pointer to a new zero value
// for the specified type. That is, the returned Value's Type is PointerTo(typ).
func New(typ Type) Value {
	if typ == nil {
		panic("reflect: New(nil)")
	}
	t := &typ.(*rtype).t
	if t.Kind() == abi.Func {
		t = closureOf(t.FuncType())
	}
	pt := ptrTo(t)
	if ifaceIndir(pt) {
		// This is a pointer to a not-in-heap type.
		panic("reflect: New of type that may not be allocated in heap (possibly undefined cgo C type)")
	}
	ptr := unsafe_New(t)
	fl := flag(Pointer)
	return Value{pt, ptr, fl}
}

// NewAt returns a Value representing a pointer to a value of the
// specified type, using p as that pointer.
func NewAt(typ Type, p unsafe.Pointer) Value {
	fl := flag(Pointer)
	t := typ.(*rtype)
	if t.Kind() == Func {
		t = toRType(closureOf(t.t.FuncType()))
	}
	return Value{t.ptrTo(), p, fl}
}

// assignTo returns a value v that can be assigned directly to dst.
// It panics if v is not assignable to dst.
// For a conversion to an interface type, target, if not nil,
// is a suggested scratch space to use.
// target must be initialized memory (or nil).
func (v Value) assignTo(context string, dst *abi.Type, target unsafe.Pointer) Value {
	if v.flag&flagMethod != 0 {
		v = makeMethodValue(context, v)
	}

	switch {
	case directlyAssignable(dst, v.typ()):
		// Overwrite type so that they match.
		// Same memory layout, so no harm done.
		fl := v.flag&(flagAddr|flagIndir) | v.flag.ro()
		fl |= flag(dst.Kind())
		return Value{dst, v.ptr, fl}

	case implements(dst, v.typ()):
		if v.Kind() == Interface && v.IsNil() {
			// A nil ReadWriter passed to nil Reader is OK,
			// but using ifaceE2I below will panic.
			// Avoid the panic by returning a nil dst (e.g., Reader) explicitly.
			return Value{dst, nil, flag(Interface)}
		}
		x := valueInterface(v, false)
		if target == nil {
			target = unsafe_New(dst)
		}
		if dst.NumMethod() == 0 {
			*(*any)(target) = x
		} else {
			ifaceE2I(dst, x, target)
		}
		return Value{dst, target, flagIndir | flag(Interface)}
	}

	// Failed.
	panic(context + ": value of type " + stringFor(v.typ()) + " is not assignable to type " + stringFor(dst))
}

// Convert returns the value v converted to type t.
// If the usual Go conversion rules do not allow conversion
// of the value v to type t, or if converting v to type t panics, Convert panics.
func (v Value) Convert(t Type) Value {
	if v.flag&flagMethod != 0 {
		v = makeMethodValue("Convert", v)
	}
	var op func(Value, Type) Value
	if kind := v.Kind(); kind == Func && kind == t.Kind() {
		op = convertOp(t.common(), v.Type().common())
	} else {
		op = convertOp(t.common(), v.typ())
	}
	if op == nil {
		panic("reflect.Value.Convert: value of type " + stringFor(v.typ()) + " cannot be converted to type " + t.String())
	}
	return op(v, t)
}

// CanConvert reports whether the value v can be converted to type t.
// If v.CanConvert(t) returns true then v.Convert(t) will not panic.
func (v Value) CanConvert(t Type) bool {
	vt := v.Type()
	if !vt.ConvertibleTo(t) {
		return false
	}
	// Converting from slice to array or to pointer-to-array can panic
	// depending on the value.
	switch {
	case vt.Kind() == Slice && t.Kind() == Array:
		if t.Len() > v.Len() {
			return false
		}
	case vt.Kind() == Slice && t.Kind() == Pointer && t.Elem().Kind() == Array:
		n := t.Elem().Len()
		if n > v.Len() {
			return false
		}
	}
	return true
}

// memmove copies size bytes to dst from src. No write barriers are used.
//
//go:linkname memmove C.memmove
func memmove(dst, src unsafe.Pointer, size uintptr)

// typedmemmove copies a value of type t to dst from src.
//
//go:linkname typedmemmove github.com/goplus/llgo/runtime/internal/runtime.Typedmemmove
func typedmemmove(t *abi.Type, dst, src unsafe.Pointer)

// typedmemclr zeros the value at ptr of type t.
//
//go:linkname typedmemclr github.com/goplus/llgo/runtime/internal/runtime.Typedmemclr
func typedmemclr(t *abi.Type, ptr unsafe.Pointer)

// typedslicecopy copies a slice of elemType values from src to dst,
// returning the number of elements copied.
func typedslicecopy(t *abi.Type, dst, src unsafeheaderSlice) int {
	rdst := *(*runtime.Slice)(unsafe.Pointer(&dst))
	rsrc := *(*runtime.Slice)(unsafe.Pointer(&src))
	return runtime.Typedslicecopy(t, rdst, rsrc)
}

/*
	TODO(xsw):

// typedmemclrpartial is like typedmemclr but assumes that
// dst points off bytes into the value and only clears size bytes.
//
//go:noescape
func typedmemclrpartial(t *abi.Type, ptr unsafe.Pointer, off, size uintptr)

// typedarrayclear zeroes the value at ptr of an array of elemType,
// only clears len elem.
//
//go:noescape
func typedarrayclear(elemType *abi.Type, ptr unsafe.Pointer, len int)

//go:noescape
func typehash(t *abi.Type, p unsafe.Pointer, h uintptr) uintptr
*/
func verifyNotInHeapPtr(p uintptr) bool {
	return true
}

func growslice(src []byte, num, etSize int) []byte {
	rs := *(*runtime.Slice)(unsafe.Pointer(&src))
	rs = runtime.GrowSlice(rs, num, etSize)
	return *(*[]byte)(unsafe.Pointer(&rs))
}

// Dummy annotation marking that the value x escapes,
// for use in cases where the reflect code is so clever that
// the compiler cannot follow.
func escapes(x any) {
	if dummy.b {
		dummy.x = x
	}
}

var dummy struct {
	b bool
	x any
}

// Dummy annotation marking that the content of value x
// escapes (i.e. modeling roughly heap=*x),
// for use in cases where the reflect code is so clever that
// the compiler cannot follow.
func contentEscapes(x unsafe.Pointer) {
	if dummy.b {
		escapes(*(*any)(x)) // the dereference may not always be safe, but never executed
	}
}

//go:nosplit
func noescape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}

// Method returns a function value corresponding to v's i'th method.
// The arguments to a Call on the returned function should not include
// a receiver; the returned function will always use v as the receiver.
// Method panics if i is out of range or if v is a nil interface value.
func (v Value) Method(i int) Value {
	if v.typ() == nil {
		panic(&ValueError{"reflect.Value.Method", Invalid})
	}
	if v.flag&flagMethod != 0 || uint(i) >= uint(toRType(v.typ()).NumMethod()) {
		panic("reflect: Method index out of range")
	}
	if v.typ().Kind() == abi.Interface && v.IsNil() {
		panic("reflect: Method on nil interface value")
	}
	fl := v.flag.ro() | (v.flag & flagIndir)
	fl |= flag(Func)
	fl |= flag(i)<<flagMethodShift | flagMethod
	return Value{v.typ(), v.ptr, fl}
}

// NumMethod returns the number of methods in the value's method set.
//
// For a non-interface type, it returns the number of exported methods.
//
// For an interface type, it returns the number of exported and unexported methods.
func (v Value) NumMethod() int {
	if v.typ() == nil {
		panic(&ValueError{"reflect.Value.NumMethod", Invalid})
	}
	if v.flag&flagMethod != 0 {
		return 0
	}
	return toRType(v.typ()).NumMethod()
}

// MethodByName returns a function value corresponding to the method
// of v with the given name.
// The arguments to a Call on the returned function should not include
// a receiver; the returned function will always use v as the receiver.
// It returns the zero Value if no method was found.
func (v Value) MethodByName(name string) Value {
	if v.typ() == nil {
		panic(&ValueError{"reflect.Value.MethodByName", Invalid})
	}
	if v.flag&flagMethod != 0 {
		return Value{}
	}
	m, ok := toRType(v.typ()).MethodByName(name)
	if !ok {
		return Value{}
	}
	return v.Method(m.Index)
}

// NumField returns the number of fields in the struct v.
// It panics if v's Kind is not Struct.
func (v Value) NumField() int {
	v.mustBe(Struct)
	tt := (*structType)(unsafe.Pointer(v.typ()))
	return len(tt.Fields)
}

// Call calls the function v with the input arguments in.
// For example, if len(in) == 3, v.Call(in) represents the Go call v(in[0], in[1], in[2]).
// Call panics if v's Kind is not Func.
// It returns the output results as Values.
// As in Go, each input argument must be assignable to the
// type of the function's corresponding input parameter.
// If v is a variadic function, Call creates the variadic slice parameter
// itself, copying in the corresponding values.
func (v Value) Call(in []Value) []Value {
	v.mustBe(Func)
	v.mustBeExported()
	return v.call("Call", in)
}

// CallSlice calls the variadic function v with the input arguments in,
// assigning the slice in[len(in)-1] to v's final variadic argument.
// For example, if len(in) == 3, v.CallSlice(in) represents the Go call v(in[0], in[1], in[2]...).
// CallSlice panics if v's Kind is not Func or if v is not variadic.
// It returns the output results as Values.
// As in Go, each input argument must be assignable to the
// type of the function's corresponding input parameter.
func (v Value) CallSlice(in []Value) []Value {
	v.mustBe(Func)
	v.mustBeExported()
	return v.call("CallSlice", in)
}

type closure struct {
	fn  unsafe.Pointer
	env unsafe.Pointer
}

func toFFIWordArg(v Value) unsafe.Pointer {
	if v.flag&flagIndir != 0 {
		return v.ptr
	}
	return unsafe.Pointer(&v.ptr)
}

func toFFIArg(v Value, typ *abi.Type) unsafe.Pointer {
	kind := typ.Kind()
	switch kind {
	case abi.Bool, abi.Int, abi.Int8, abi.Int16, abi.Int32, abi.Int64,
		abi.Uint, abi.Uint8, abi.Uint16, abi.Uint32, abi.Uint64, abi.Uintptr,
		abi.Float32, abi.Float64:
		if v.flag&flagIndir != 0 {
			return v.ptr
		} else {
			return unsafe.Pointer(&v.ptr)
		}
	case abi.Complex64, abi.Complex128:
		return unsafe.Pointer(v.ptr)
	case abi.Array:
		if v.flag&flagIndir != 0 {
			return v.ptr
		}
		return unsafe.Pointer(&v.ptr)
	case abi.Chan:
		return toFFIWordArg(v)
	case abi.Func:
		if v.flag&flagIndir != 0 {
			return v.ptr
		}
		return unsafe.Pointer(&v.ptr)
	case abi.Interface:
		i := v.Interface()
		return unsafe.Pointer(&i)
	case abi.Map:
		return toFFIWordArg(v)
	case abi.Pointer:
		return toFFIWordArg(v)
	case abi.Slice:
		return v.ptr
	case abi.String:
		return v.ptr
	case abi.Struct:
		if v.flag&flagIndir != 0 {
			return v.ptr
		}
		return unsafe.Pointer(&v.ptr)
	case abi.UnsafePointer:
		return toFFIWordArg(v)
	}
	panic("reflect.toFFIArg unsupport type " + v.typ().String())
}

var (
	ffiTypeClosure = ffi.StructOf(ffi.TypePointer, ffi.TypePointer)
)

func toFFIType(typ *abi.Type) *ffi.Type {
	kind := typ.Kind()
	switch kind {
	case abi.Bool, abi.Int, abi.Int8, abi.Int16, abi.Int32, abi.Int64,
		abi.Uint, abi.Uint8, abi.Uint16, abi.Uint32, abi.Uint64, abi.Uintptr,
		abi.Float32, abi.Float64, abi.Complex64, abi.Complex128:
		return ffi.Typ[kind]
	case abi.Array:
		st := typ.ArrayType()
		return ffi.ArrayOf(toFFIType(st.Elem), int(st.Len))
	case abi.Chan:
		return ffi.TypePointer
	case abi.Func:
		return ffiTypeClosure
	case abi.Interface:
		return ffi.TypeInterface
	case abi.Map:
		return ffi.TypePointer
	case abi.Pointer:
		return ffi.TypePointer
	case abi.Slice:
		return ffi.TypeSlice
	case abi.String:
		return ffi.TypeString
	case abi.Struct:
		return toFFIStructType(typ)
	case abi.UnsafePointer:
		return ffi.TypePointer
	}
	panic("reflect.toFFIType unsupport type " + typ.String())
}

func toFFIStructType(typ *abi.Type) *ffi.Type {
	st := typ.StructType()
	fields := make([]*ffi.Type, 0, len(st.Fields))
	var off uintptr
	for _, fs := range st.Fields {
		if fs.Offset > off {
			fields, off = appendFFIPadding(fields, off, fs.Offset-off)
		}
		if fs.Typ.Size_ == 0 {
			continue
		}
		fields = append(fields, toFFIType(fs.Typ))
		off = fs.Offset + fs.Typ.Size_
	}
	// Do not pad to typ.Size_: trailing zero-sized fields can enlarge the
	// Go-visible size without consuming registers in llgo's callable ABI.
	return ffi.StructOf(fields...)
}

func appendFFIPadding(fields []*ffi.Type, off, size uintptr) ([]*ffi.Type, uintptr) {
	for size > 0 {
		switch {
		case off%8 == 0 && size >= 8:
			fields = append(fields, ffi.TypeUint64)
			off += 8
			size -= 8
		case off%4 == 0 && size >= 4:
			fields = append(fields, ffi.TypeUint32)
			off += 4
			size -= 4
		case off%2 == 0 && size >= 2:
			fields = append(fields, ffi.TypeUint16)
			off += 2
			size -= 2
		default:
			fields = append(fields, ffi.TypeUint8)
			off++
			size--
		}
	}
	return fields, off
}

func toFFISig(tin, tout []*abi.Type) (*ffi.Signature, error) {
	args := make([]*ffi.Type, len(tin))
	for i, in := range tin {
		args[i] = toFFIType(in)
	}
	return ffi.NewSignature(toFFIRetType(tout), args...)
}

func toFFIRetType(tout []*abi.Type) *ffi.Type {
	switch n := len(tout); n {
	case 0:
		return ffi.TypeVoid
	case 1:
		return toFFIType(tout[0])
	default:
		fields := make([]*ffi.Type, n)
		for i, out := range tout {
			fields[i] = toFFIType(out)
		}
		return ffi.StructOf(fields...)
	}
}

func (v Value) closureFunc() *abi.FuncType {
	return toFuncType(v.typ_.StructType())
}

func (v Value) call(op string, in []Value) (out []Value) {
	var (
		ft   *abi.FuncType
		tin  []*abi.Type
		tout []*abi.Type
		args []unsafe.Pointer
		fn   unsafe.Pointer
		ret  unsafe.Pointer
		ioff int
	)
	if v.typ_.IsClosure() && v.flag&flagMethod == 0 {
		ft = v.typ_.StructType().Fields[0].Typ.FuncType()
		tin = append([]*abi.Type{rtypeOf(unsafe.Pointer(nil))}, ft.In...)
		tout = ft.Out
		c := (*struct {
			fn  unsafe.Pointer
			env unsafe.Pointer
		})(v.ptr)
		fn = c.fn
		ioff = 1
		args = append(args, unsafe.Pointer(&c.env))
	} else {
		if v.flag&flagMethod != 0 {
			var (
				rcvrtype *abi.Type
			)
			rcvrtype, ft, fn = methodReceiver(op, v, int(v.flag)>>flagMethodShift)
			tin = append([]*abi.Type{rcvrtype}, ft.In...)
			tout = ft.Out
			ioff = 1
			var ptr unsafe.Pointer
			storeRcvr(v, unsafe.Pointer(&ptr))
			args = append(args, unsafe.Pointer(&ptr))
		} else {
			if v.flag&flagIndir != 0 {
				fn = *(*unsafe.Pointer)(v.ptr)
			} else {
				fn = v.ptr
			}
			ft = v.typ_.FuncType()
			tin = ft.In
			tout = ft.Out
		}
	}

	isSlice := op == "CallSlice"
	n := len(ft.In)
	isVariadic := ft.Variadic()
	if isSlice {
		if !isVariadic {
			panic("reflect: CallSlice of non-variadic function")
		}
		if len(in) < n {
			panic("reflect: CallSlice with too few input arguments")
		}
		if len(in) > n {
			panic("reflect: CallSlice with too many input arguments")
		}
	} else {
		if isVariadic {
			n--
		}
		if len(in) < n {
			panic("reflect: Call with too few input arguments")
		}
		if !isVariadic && len(in) > n {
			panic("reflect: Call with too many input arguments")
		}
	}
	for _, x := range in {
		if x.Kind() == Invalid {
			panic("reflect: " + op + " using zero Value argument")
		}
	}
	for i := 0; i < n; i++ {
		if xt, targ := in[i].Type(), ft.In[i]; !xt.AssignableTo(toRType(targ)) {
			panic("reflect: " + op + " using " + xt.String() + " as type " + stringFor(targ))
		}
	}
	if !isSlice && isVariadic {
		// prepare slice for remaining values
		m := len(in) - n
		slice := MakeSlice(toRType(ft.In[n]), m, m)
		elem := toRType(ft.In[n].Elem()) // FIXME cast to slice type and Elem()
		for i := 0; i < m; i++ {
			x := in[n+i]
			if xt := x.Type(); !xt.AssignableTo(elem) {
				panic("reflect: cannot use " + xt.String() + " as type " + elem.String() + " in " + op)
			}
			slice.Index(i).Set(x)
		}
		origIn := in
		in = make([]Value, n+1)
		copy(in[:n], origIn)
		in[n] = slice
	}

	nin := len(in)
	if nin != len(ft.In) {
		panic("reflect.Value.Call: wrong argument count")
	}

	ffiArgs := make([]*ffi.Type, 0, len(tin)+4)
	for i := 0; i < ioff; i++ {
		ffiArgs = append(ffiArgs, toFFIType(tin[i]))
	}
	for i, arg := range in {
		typ := tin[ioff+i]
		if ffiCallSliceAsTriple && typ.Kind() == abi.Slice {
			h := (*unsafeheaderSlice)(arg.ptr)
			ffiArgs = append(ffiArgs, ffi.TypePointer, ffi.TypeInt, ffi.TypeInt)
			args = append(args, unsafe.Pointer(&h.Data), unsafe.Pointer(&h.Len), unsafe.Pointer(&h.Cap))
			continue
		}
		ffiArgs = append(ffiArgs, toFFIType(typ))
		args = append(args, toFFIArg(arg, typ))
	}

	sig, err := ffi.NewSignature(toFFIRetType(tout), ffiArgs...)
	if err != nil {
		panic(err)
	}
	if sig.RType != ffi.TypeVoid {
		v := runtime.AllocZ(sig.RType.Size)
		ret = unsafe.Pointer(&v)
	}

	ffi.Call(sig, fn, ret, args...)
	switch n := len(tout); n {
	case 0:
	case 1:
		out := NewAt(toType(tout[0]), ret).Elem()
		resolveIndirectValue(&out, tout[0])
		return []Value{out}
	default:
		out = make([]Value, n)
		alignment := uintptr(sig.RType.Alignment)
		var off uintptr
		for i, tout := range tout {
			out[i] = NewAt(toType(tout), add(ret, off, "")).Elem()
			resolveIndirectValue(&out[i], tout)
			off += (tout.Size_ + alignment - 1) &^ (alignment - 1)
		}
	}
	return
}

// resolveIndirectValue converts Value's indirect reference to direct based on abi.Type, clears flagIndir
// It adjusts the Value's ptr (dereferences or truncates) and removes the flagIndir flag.
func resolveIndirectValue(v *Value, typ *abi.Type) {
	if !typ.IsDirectIface() || v.flag&flagIndir == 0 {
		return
	}
	k := typ.Kind()
	switch {
	case k == abi.Float32:
		fv := *(*float32)(v.ptr)
		v.ptr = unsafe.Pointer(uintptr(bitcast.FromFloat32(fv)))
	case k == abi.Float64:
		fv := *(*float64)(v.ptr)
		v.ptr = unsafe.Pointer(uintptr(bitcast.FromFloat64(fv)))
	default:
		// convert abi.Bool~abi.Uintptr and abi.Struct/abi.Array themselves.
		// struct/array elem conversion by caller func.
		if typ.Size_ < unsafe.Sizeof(0) {
			v.ptr = truncate(*(*unsafe.Pointer)(v.ptr), typ.Size_*8)
		} else {
			v.ptr = *(*unsafe.Pointer)(v.ptr)
		}
	}
	v.flag &= ^flagIndir
}

func truncate(addr unsafe.Pointer, bits uintptr) unsafe.Pointer {
	mask := uintptr(1<<bits) - 1
	return unsafe.Pointer(uintptr(addr) & mask)
}

// v is a method receiver. Store at p the word which is used to
// encode that receiver at the start of the argument list.
// Reflect uses the "interface" calling convention for
// methods, which always uses one word to record the receiver.
func storeRcvr(v Value, p unsafe.Pointer) {
	t := v.typ()
	if t.Kind() == abi.Interface {
		// the interface data word becomes the receiver word
		iface := (*nonEmptyInterface)(v.ptr)
		*(*unsafe.Pointer)(p) = ifacePtrData(iface)
	} else if v.flag&flagIndir != 0 && !ifaceIndir(t) {
		*(*unsafe.Pointer)(p) = *(*unsafe.Pointer)(v.ptr)
	} else if v.flag&flagIndir == 0 && runtime.DirectIfaceData(t) {
		*(*unsafe.Pointer)(p) = unsafe.Pointer(&v.ptr)
	} else {
		*(*unsafe.Pointer)(p) = v.ptr
	}
}

func ifacePtrData(i *nonEmptyInterface) unsafe.Pointer {
	if runtime.DirectIfaceData(i.itab.typ) {
		return unsafe.Pointer(&i.word)
	}
	return i.word
}

var stringType = rtypeOf("")

// MapIndex returns the value associated with key in the map v.
// It panics if v's Kind is not Map.
// It returns the zero Value if key is not found in the map or if v represents a nil map.
// As in Go, the key's value must be assignable to the map's key type.
func (v Value) MapIndex(key Value) Value {
	v.mustBe(Map)
	tt := (*mapType)(unsafe.Pointer(v.typ()))

	// Do not require key to be exported, so that DeepEqual
	// and other programs can use all the keys returned by
	// MapKeys as arguments to MapIndex. If either the map
	// or the key is unexported, though, the result will be
	// considered unexported. This is consistent with the
	// behavior for structs, which allow read but not write
	// of unexported fields.

	var e unsafe.Pointer
	// if (tt.Key == stringType || key.kind() == String) && tt.Key == key.typ() && tt.Elem.Size() <= maxValSize {
	// 	k := *(*string)(key.ptr)
	// 	e = mapaccess_faststr(v.typ(), v.pointer(), k)
	// } else {
	key = key.assignTo("reflect.Value.MapIndex", tt.Key, nil)
	var k unsafe.Pointer
	if key.flag&flagIndir != 0 {
		k = key.ptr
	} else {
		k = unsafe.Pointer(&key.ptr)
	}
	var ok bool
	e, ok = mapaccess(v.typ(), v.pointer(), k)
	if !ok {
		return Value{}
	}
	typ := tt.Elem
	fl := (v.flag | key.flag).ro()
	fl |= flag(typ.Kind())
	return copyVal(typ, fl, e)
}

// MapKeys returns a slice containing all the keys present in the map,
// in unspecified order.
// It panics if v's Kind is not Map.
// It returns an empty slice if v represents a nil map.
func (v Value) MapKeys() []Value {
	v.mustBe(Map)
	tt := (*mapType)(unsafe.Pointer(v.typ()))
	keyType := tt.Key

	fl := v.flag.ro() | flag(keyType.Kind())

	m := v.pointer()
	mlen := int(0)
	if m != nil {
		mlen = maplen(m)
	}
	var it hiter
	mapiterinit(v.typ(), m, &it)
	a := make([]Value, mlen)
	var i int
	for i = 0; i < len(a); i++ {
		key := mapiterkey(&it)
		if key == nil {
			// Someone deleted an entry from the map since we
			// called maplen above. It's a data race, but nothing
			// we can do about it.
			break
		}
		a[i] = copyVal(keyType, fl, key)
		mapiternext(&it)
	}
	return a[:i]
}

// hiter's structure matches runtime.hiter's structure.
// Having a clone here allows us to embed a map iterator
// inside type MapIter so that MapIters can be re-used
// without doing any allocations.
type hiter struct {
	key         unsafe.Pointer
	elem        unsafe.Pointer
	t           unsafe.Pointer
	h           unsafe.Pointer
	buckets     unsafe.Pointer
	bptr        unsafe.Pointer
	overflow    *[]unsafe.Pointer
	oldoverflow *[]unsafe.Pointer
	startBucket uintptr
	offset      uint8
	wrapped     bool
	B           uint8
	i           uint8
	bucket      uintptr
	checkBucket uintptr
}

func (h *hiter) initialized() bool {
	return h.t != nil
}

// A MapIter is an iterator for ranging over a map.
// See Value.MapRange.
type MapIter struct {
	m     Value
	hiter hiter
}

// Key returns the key of iter's current map entry.
func (iter *MapIter) Key() Value {
	if !iter.hiter.initialized() {
		panic("MapIter.Key called before Next")
	}
	iterkey := mapiterkey(&iter.hiter)
	if iterkey == nil {
		panic("MapIter.Key called on exhausted iterator")
	}

	t := (*mapType)(unsafe.Pointer(iter.m.typ()))
	ktype := t.Key
	return copyVal(ktype, iter.m.flag.ro()|flag(ktype.Kind()), iterkey)
}

// SetIterKey assigns to v the key of iter's current map entry.
// It is equivalent to v.Set(iter.Key()), but it avoids allocating a new Value.
// As in Go, the key must be assignable to v's type and
// must not be derived from an unexported field.
func (v Value) SetIterKey(iter *MapIter) {
	if !iter.hiter.initialized() {
		panic("reflect: Value.SetIterKey called before Next")
	}
	iterkey := mapiterkey(&iter.hiter)
	if iterkey == nil {
		panic("reflect: Value.SetIterKey called on exhausted iterator")
	}

	v.mustBeAssignable()
	var target unsafe.Pointer
	if v.kind() == Interface {
		target = v.ptr
	}

	t := (*mapType)(unsafe.Pointer(iter.m.typ()))
	ktype := t.Key

	iter.m.mustBeExported() // do not let unexported m leak
	key := Value{ktype, iterkey, iter.m.flag | flag(ktype.Kind()) | flagIndir}
	key = key.assignTo("reflect.MapIter.SetKey", v.typ(), target)
	typedmemmove(v.typ(), v.ptr, key.ptr)
}

// Value returns the value of iter's current map entry.
func (iter *MapIter) Value() Value {
	if !iter.hiter.initialized() {
		panic("MapIter.Value called before Next")
	}
	iterelem := mapiterelem(&iter.hiter)
	if iterelem == nil {
		panic("MapIter.Value called on exhausted iterator")
	}

	t := (*mapType)(unsafe.Pointer(iter.m.typ()))
	vtype := t.Elem
	return copyVal(vtype, iter.m.flag.ro()|flag(vtype.Kind()), iterelem)
}

// SetIterValue assigns to v the value of iter's current map entry.
// It is equivalent to v.Set(iter.Value()), but it avoids allocating a new Value.
// As in Go, the value must be assignable to v's type and
// must not be derived from an unexported field.
func (v Value) SetIterValue(iter *MapIter) {
	if !iter.hiter.initialized() {
		panic("reflect: Value.SetIterValue called before Next")
	}
	iterelem := mapiterelem(&iter.hiter)
	if iterelem == nil {
		panic("reflect: Value.SetIterValue called on exhausted iterator")
	}

	v.mustBeAssignable()
	var target unsafe.Pointer
	if v.kind() == Interface {
		target = v.ptr
	}

	t := (*mapType)(unsafe.Pointer(iter.m.typ()))
	vtype := t.Elem

	iter.m.mustBeExported() // do not let unexported m leak
	elem := Value{vtype, iterelem, iter.m.flag | flag(vtype.Kind()) | flagIndir}
	elem = elem.assignTo("reflect.MapIter.SetValue", v.typ(), target)
	typedmemmove(v.typ(), v.ptr, elem.ptr)
}

// Next advances the map iterator and reports whether there is another
// entry. It returns false when iter is exhausted; subsequent
// calls to Key, Value, or Next will panic.
func (iter *MapIter) Next() bool {
	if !iter.m.IsValid() {
		panic("MapIter.Next called on an iterator that does not have an associated map Value")
	}
	if !iter.hiter.initialized() {
		mapiterinit(iter.m.typ(), iter.m.pointer(), &iter.hiter)
	} else {
		if mapiterkey(&iter.hiter) == nil {
			panic("MapIter.Next called on exhausted iterator")
		}
		mapiternext(&iter.hiter)
	}
	return mapiterkey(&iter.hiter) != nil
}

// Reset modifies iter to iterate over v.
// It panics if v's Kind is not Map and v is not the zero Value.
// Reset(Value{}) causes iter to not to refer to any map,
// which may allow the previously iterated-over map to be garbage collected.
func (iter *MapIter) Reset(v Value) {
	if v.IsValid() {
		v.mustBe(Map)
	}
	iter.m = v
	iter.hiter = hiter{}
}

// MapRange returns a range iterator for a map.
// It panics if v's Kind is not Map.
//
// Call Next to advance the iterator, and Key/Value to access each entry.
// Next returns false when the iterator is exhausted.
// MapRange follows the same iteration semantics as a range statement.
//
// Example:
//
//	iter := reflect.ValueOf(m).MapRange()
//	for iter.Next() {
//		k := iter.Key()
//		v := iter.Value()
//		...
//	}
func (v Value) MapRange() *MapIter {
	// This is inlinable to take advantage of "function outlining".
	// The allocation of MapIter can be stack allocated if the caller
	// does not allow it to escape.
	// See https://blog.filippo.io/efficient-go-apis-with-the-inliner/
	if v.kind() != Map {
		v.panicNotMap()
	}
	return &MapIter{m: v}
}

// Force slow panicking path not inlined, so it won't add to the
// inlining budget of the caller.
// TODO undo when the inliner is no longer bottom-up only.
//
//go:noinline
func (f flag) panicNotMap() {
	f.mustBe(Map)
}

// copyVal returns a Value containing the map key or value at ptr,
// allocating a new variable as needed.
func copyVal(typ *abi.Type, fl flag, ptr unsafe.Pointer) Value {
	if typ.IfaceIndir() {
		// Copy result so future changes to the map
		// won't change the underlying value.
		c := unsafe_New(typ)
		typedmemmove(typ, c, ptr)
		return Value{typ, c, fl | flagIndir}
	}
	return Value{typ, *(*unsafe.Pointer)(ptr), fl}
}

// methodReceiver returns information about the receiver
// described by v. The Value v may or may not have the
// flagMethod bit set, so the kind cached in v.flag should
// not be used.
// The return value rcvrtype gives the method's actual receiver type.
// The return value t gives the method type signature (without the receiver).
// The return value fn is a pointer to the method code.
func methodReceiver(op string, v Value, methodIndex int) (rcvrtype *abi.Type, t *funcType, fn unsafe.Pointer) {
	i := methodIndex
	if v.typ().Kind() == abi.Interface {
		tt := (*interfaceType)(unsafe.Pointer(v.typ()))
		if uint(i) >= uint(len(tt.Methods)) {
			panic("reflect: internal error: invalid method index")
		}
		m := &tt.Methods[i]
		if !abi.IsExported(m.Name()) {
			panic("reflect: " + op + " of unexported method")
		}
		iface := (*nonEmptyInterface)(v.ptr)
		if iface.itab == nil {
			panic("reflect: " + op + " of method on nil interface value")
		}
		rcvrtype = iface.itab.typ
		fn = unsafe.Pointer(iface.itab.fun[i])
		t = (*funcType)(unsafe.Pointer(m.Typ_))
	} else {
		rcvrtype = v.typ()
		ms := v.typ().ExportedMethods()
		if uint(i) >= uint(len(ms)) {
			panic("reflect: internal error: invalid method index")
		}
		m := ms[i]
		if !abi.IsExported(m.Name()) {
			panic("reflect: " + op + " of unexported method")
		}
		fn = unsafe.Pointer(m.Ifn_)
		t = (*funcType)(unsafe.Pointer(m.Mtyp_))
	}
	if rcvrtype.Kind() != abi.Pointer {
		// convert rcvrtype to abi.Pointer to match ifn call
		rcvrtype = toRType(rcvrtype).ptrTo()
	}
	return
}

// convertOp returns the function to convert a value of type src
// to a value of type dst. If the conversion is illegal, convertOp returns nil.
func convertOp(dst, src *abi.Type) func(Value, Type) Value {
	switch Kind(src.Kind()) {
	case Int, Int8, Int16, Int32, Int64:
		switch Kind(dst.Kind()) {
		case Int, Int8, Int16, Int32, Int64, Uint, Uint8, Uint16, Uint32, Uint64, Uintptr:
			return cvtInt
		case Float32, Float64:
			return cvtIntFloat
		case String:
			return cvtIntString
		}

	case Uint, Uint8, Uint16, Uint32, Uint64, Uintptr:
		switch Kind(dst.Kind()) {
		case Int, Int8, Int16, Int32, Int64, Uint, Uint8, Uint16, Uint32, Uint64, Uintptr:
			return cvtUint
		case Float32, Float64:
			return cvtUintFloat
		case String:
			return cvtUintString
		}

	case Float32, Float64:
		switch Kind(dst.Kind()) {
		case Int, Int8, Int16, Int32, Int64:
			return cvtFloatInt
		case Uint, Uint8, Uint16, Uint32, Uint64, Uintptr:
			return cvtFloatUint
		case Float32, Float64:
			return cvtFloat
		}

	case Complex64, Complex128:
		switch Kind(dst.Kind()) {
		case Complex64, Complex128:
			return cvtComplex
		}

	case String:
		if dst.Kind() == abi.Slice && pkgPathFor(dst.Elem()) == "" {
			switch Kind(dst.Elem().Kind()) {
			case Uint8:
				return cvtStringBytes
			case Int32:
				return cvtStringRunes
			}
		}

	case Slice:
		if dst.Kind() == abi.String && pkgPathFor(src.Elem()) == "" {
			switch Kind(src.Elem().Kind()) {
			case Uint8:
				return cvtBytesString
			case Int32:
				return cvtRunesString
			}
		}
		// "x is a slice, T is a pointer-to-array type,
		// and the slice and array types have identical element types."
		if dst.Kind() == abi.Pointer && dst.Elem().Kind() == abi.Array && src.Elem() == dst.Elem().Elem() {
			return cvtSliceArrayPtr
		}
		// "x is a slice, T is an array type,
		// and the slice and array types have identical element types."
		if dst.Kind() == abi.Array && src.Elem() == dst.Elem() {
			return cvtSliceArray
		}

	case Chan:
		if dst.Kind() == abi.Chan && specialChannelAssignability(dst, src) {
			return cvtDirect
		}
	}

	// dst and src have same underlying type.
	if haveIdenticalUnderlyingType(dst, src, false) {
		return cvtDirect
	}

	// dst and src are non-defined pointer types with same underlying base type.
	if dst.Kind() == abi.Pointer && nameFor(dst) == "" &&
		src.Kind() == abi.Pointer && nameFor(src) == "" &&
		haveIdenticalUnderlyingType(elem(dst), elem(src), false) {
		return cvtDirect
	}

	if implements(dst, src) {
		if src.Kind() == abi.Interface {
			return cvtI2I
		}
		return cvtT2I
	}

	return nil
}

// _64bit = true on 64-bit systems, false on 32-bit systems
const is64bit = (1 << (^uintptr(0) >> 63) / 2) == 1

// makeInt returns a Value of type t equal to bits (possibly truncated),
// where t is a signed or unsigned int type.
func makeInt(f flag, bits uint64, t Type) Value {
	typ := t.common()
	var ptr unsafe.Pointer
	switch typ.Size() {
	case 1, 2, 4:
		ptr = unsafe.Pointer(uintptr(bits))
	case 8:
		if is64bit {
			ptr = unsafe.Pointer(uintptr(bits))
		} else {
			ptr = unsafe_New(typ)
			*(*uint64)(ptr) = bits
			f |= flagIndir
		}
	}
	return Value{typ, ptr, f | flag(typ.Kind())}
}

// makeFloat returns a Value of type t equal to v (possibly truncated to float32),
// where t is a float32 or float64 type.
func makeFloat(f flag, v float64, t Type) Value {
	typ := t.common()
	var ptr unsafe.Pointer
	switch typ.Size() {
	case 4:
		ptr = unsafe.Pointer(uintptr(bitcast.FromFloat32(float32(v))))
	case 8:
		if is64bit {
			ptr = unsafe.Pointer(uintptr(bitcast.FromFloat64(v)))
		} else {
			ptr = unsafe_New(typ)
			*(*float64)(ptr) = v
			f |= flagIndir
		}
	}
	return Value{typ, ptr, f | flag(typ.Kind())}
}

// makeFloat32 returns a Value of type t equal to v, where t is a float32 type.
func makeFloat32(f flag, ptr unsafe.Pointer, t Type) Value {
	typ := t.common()
	return Value{typ, ptr, f | flag(typ.Kind())}
}

// makeComplex returns a Value of type t equal to v (possibly truncated to complex64),
// where t is a complex64 or complex128 type.
func makeComplex(f flag, v complex128, t Type) Value {
	typ := t.common()
	ptr := unsafe_New(typ)
	switch typ.Size() {
	case 8:
		*(*complex64)(ptr) = complex64(v)
	case 16:
		*(*complex128)(ptr) = v
	}
	return Value{typ, ptr, f | flagIndir | flag(typ.Kind())}
}

func makeString(f flag, v string, t Type) Value {
	ret := New(t).Elem()
	ret.SetString(v)
	ret.flag = ret.flag&^flagAddr | f
	return ret
}

func makeBytes(f flag, v []byte, t Type) Value {
	ret := New(t).Elem()
	ret.SetBytes(v)
	ret.flag = ret.flag&^flagAddr | f
	return ret
}

func makeRunes(f flag, v []rune, t Type) Value {
	ret := New(t).Elem()
	ret.setRunes(v)
	ret.flag = ret.flag&^flagAddr | f
	return ret
}

// These conversion functions are returned by convertOp
// for classes of conversions. For example, the first function, cvtInt,
// takes any value v of signed int type and returns the value converted
// to type t, where t is any signed or unsigned int type.

// convertOp: intXX -> [u]intXX
func cvtInt(v Value, t Type) Value {
	return makeInt(v.flag.ro(), uint64(v.Int()), t)
}

// convertOp: uintXX -> [u]intXX
func cvtUint(v Value, t Type) Value {
	return makeInt(v.flag.ro(), v.Uint(), t)
}

// convertOp: floatXX -> intXX
func cvtFloatInt(v Value, t Type) Value {
	return makeInt(v.flag.ro(), uint64(int64(v.Float())), t)
}

// convertOp: floatXX -> uintXX
func cvtFloatUint(v Value, t Type) Value {
	return makeInt(v.flag.ro(), uint64(v.Float()), t)
}

// convertOp: intXX -> floatXX
func cvtIntFloat(v Value, t Type) Value {
	return makeFloat(v.flag.ro(), float64(v.Int()), t)
}

// convertOp: uintXX -> floatXX
func cvtUintFloat(v Value, t Type) Value {
	return makeFloat(v.flag.ro(), float64(v.Uint()), t)
}

// convertOp: floatXX -> floatXX
func cvtFloat(v Value, t Type) Value {
	if v.Type().Kind() == Float32 && t.Kind() == Float32 {
		// Don't do any conversion if both types have underlying type float32.
		// This avoids converting to float64 and back, which will
		// convert a signaling NaN to a quiet NaN. See issue 36400.
		return makeFloat32(v.flag.ro(), v.ptr, t)
	}
	return makeFloat(v.flag.ro(), v.Float(), t)
}

// convertOp: complexXX -> complexXX
func cvtComplex(v Value, t Type) Value {
	return makeComplex(v.flag.ro(), v.Complex(), t)
}

// convertOp: intXX -> string
func cvtIntString(v Value, t Type) Value {
	s := "\uFFFD"
	if x := v.Int(); int64(rune(x)) == x {
		s = string(rune(x))
	}
	return makeString(v.flag.ro(), s, t)
}

// convertOp: uintXX -> string
func cvtUintString(v Value, t Type) Value {
	s := "\uFFFD"
	if x := v.Uint(); uint64(rune(x)) == x {
		s = string(rune(x))
	}
	return makeString(v.flag.ro(), s, t)
}

// convertOp: []byte -> string
func cvtBytesString(v Value, t Type) Value {
	return makeString(v.flag.ro(), string(v.Bytes()), t)
}

// convertOp: string -> []byte
func cvtStringBytes(v Value, t Type) Value {
	return makeBytes(v.flag.ro(), []byte(v.String()), t)
}

// convertOp: []rune -> string
func cvtRunesString(v Value, t Type) Value {
	return makeString(v.flag.ro(), string(v.runes()), t)
}

// convertOp: string -> []rune
func cvtStringRunes(v Value, t Type) Value {
	return makeRunes(v.flag.ro(), []rune(v.String()), t)
}

// convertOp: []T -> *[N]T
func cvtSliceArrayPtr(v Value, t Type) Value {
	n := t.Elem().Len()
	if n > v.Len() {
		panic("reflect: cannot convert slice with length " + itoa(v.Len()) + " to pointer to array with length " + itoa(n))
	}
	h := (*unsafeheaderSlice)(v.ptr)
	return Value{t.common(), h.Data, v.flag&^(flagIndir|flagAddr|flagKindMask) | flag(Pointer)}
}

// convertOp: []T -> [N]T
func cvtSliceArray(v Value, t Type) Value {
	n := t.Len()
	if n > v.Len() {
		panic("reflect: cannot convert slice with length " + itoa(v.Len()) + " to array with length " + itoa(n))
	}
	h := (*unsafeheaderSlice)(v.ptr)
	typ := t.common()
	ptr := h.Data
	c := unsafe_New(typ)
	typedmemmove(typ, c, ptr)
	ptr = c

	return Value{typ, ptr, v.flag&^(flagAddr|flagKindMask) | flag(Array)}
}

// convertOp: direct copy
func cvtDirect(v Value, typ Type) Value {
	f := v.flag
	t := typ.common()
	ptr := v.ptr
	if f&flagAddr != 0 {
		// indirect, mutable word - make a copy
		c := unsafe_New(t)
		typedmemmove(t, c, ptr)
		ptr = c
		f &^= flagAddr
	}
	return Value{t, ptr, v.flag.ro() | f} // v.flag.ro()|f == f?
}

// convertOp: concrete -> interface
func cvtT2I(v Value, typ Type) Value {
	target := unsafe_New(typ.common())
	x := valueInterface(v, false)
	if typ.NumMethod() == 0 {
		*(*any)(target) = x
	} else {
		ifaceE2I(typ.common(), x, target)
	}
	return Value{typ.common(), target, v.flag.ro() | flagIndir | flag(Interface)}
}

// convertOp: interface -> interface
func cvtI2I(v Value, typ Type) Value {
	if v.IsNil() {
		ret := Zero(typ)
		ret.flag |= v.flag.ro()
		return ret
	}
	return cvtT2I(v.Elem(), typ)
}

//go:linkname chancap github.com/goplus/llgo/runtime/internal/runtime.ChanCap
func chancap(ch unsafe.Pointer) int

//go:linkname chanlen github.com/goplus/llgo/runtime/internal/runtime.ChanLen
func chanlen(ch unsafe.Pointer) int

//go:linkname makemap github.com/goplus/llgo/runtime/internal/runtime.MakeMap
func makemap(t *abi.Type, cap int) (m unsafe.Pointer)

//go:linkname maplen github.com/goplus/llgo/runtime/internal/runtime.MapLen
func maplen(ch unsafe.Pointer) int

//go:linkname mapaccess github.com/goplus/llgo/runtime/internal/runtime.MapAccess2
func mapaccess(t *abi.Type, m unsafe.Pointer, key unsafe.Pointer) (val unsafe.Pointer, ok bool)

//go:linkname mapassign0 github.com/goplus/llgo/runtime/internal/runtime.MapAssign
func mapassign0(t *abi.Type, m unsafe.Pointer, key unsafe.Pointer) unsafe.Pointer

func mapassign(t *abi.Type, m unsafe.Pointer, key, val unsafe.Pointer) {
	contentEscapes(key)
	contentEscapes(val)
	p := mapassign0(t, m, key)
	runtime.Typedmemmove(t.Elem(), p, val)
}

// //go:noescape
// func mapassign_faststr0(t *abi.Type, m unsafe.Pointer, key string, val unsafe.Pointer)

// func mapassign_faststr(t *abi.Type, m unsafe.Pointer, key string, val unsafe.Pointer) {
// 	contentEscapes((*unsafeheader.String)(unsafe.Pointer(&key)).Data)
// 	contentEscapes(val)
// 	mapassign_faststr0(t, m, key, val)
// }

//go:linkname mapdelete github.com/goplus/llgo/runtime/internal/runtime.MapDelete
func mapdelete(t *abi.Type, m unsafe.Pointer, key unsafe.Pointer)

//go:noescape
// func mapdelete_faststr(t *abi.Type, m unsafe.Pointer, key string)

//go:linkname mapiterinit github.com/goplus/llgo/runtime/internal/runtime.mapiterinit
func mapiterinit(t *abi.Type, m unsafe.Pointer, it *hiter)

func mapiterkey(it *hiter) (key unsafe.Pointer) {
	return it.key
}

func mapiterelem(it *hiter) (elem unsafe.Pointer) {
	return it.elem
}

//go:linkname mapiternext github.com/goplus/llgo/runtime/internal/runtime.mapiternext
func mapiternext(it *hiter)

//go:linkname mapclear github.com/goplus/llgo/runtime/internal/runtime.mapclear
func mapclear(t *abi.Type, m unsafe.Pointer)

//go:linkname typehash github.com/goplus/llgo/runtime/internal/runtime.typehash
func typehash(t *abi.Type, p unsafe.Pointer, h uintptr) uintptr

//go:linkname makechan github.com/goplus/llgo/runtime/internal/runtime.NewChan
func makechan(eltSize, cap int) unsafe.Pointer

// MakeSlice creates a new zero-initialized slice value
// for the specified slice type, length, and capacity.
func MakeSlice(typ Type, len, cap int) Value {
	if typ.Kind() != Slice {
		panic("reflect.MakeSlice of non-slice type")
	}
	if len < 0 {
		panic("reflect.MakeSlice: negative len")
	}
	if cap < 0 {
		panic("reflect.MakeSlice: negative cap")
	}
	if len > cap {
		panic("reflect.MakeSlice: len > cap")
	}

	s := unsafeheaderSlice{Data: unsafe_NewArray(&(typ.Elem().(*rtype).t), cap), Len: len, Cap: cap}
	return Value{&typ.(*rtype).t, unsafe.Pointer(&s), flagIndir | flag(Slice)}
}

// MakeChan creates a new channel with the specified type and buffer size.
func MakeChan(typ Type, buffer int) Value {
	if typ.Kind() != Chan {
		panic("reflect.MakeChan of non-chan type")
	}
	if buffer < 0 {
		panic("reflect.MakeChan: negative buffer size")
	}
	if typ.ChanDir() != BothDir {
		panic("reflect.MakeChan: unidirectional channel type")
	}
	t := typ.common()
	ch := makechan(int(t.Elem().Size()), buffer)
	return Value{t, ch, flag(Chan)}
}

// MakeMap creates a new map with the specified type.
func MakeMap(typ Type) Value {
	return MakeMapWithSize(typ, 0)
}

// MakeMapWithSize creates a new map with the specified type
// and initial space for approximately n elements.
func MakeMapWithSize(typ Type, n int) Value {
	if typ.Kind() != Map {
		panic("reflect.MakeMapWithSize of non-map type")
	}
	t := typ.common()
	m := makemap(t, n)
	return Value{t, m, flag(Map)}
}

// Comparable reports whether the value v is comparable.
// If the type of v is an interface, this checks the dynamic type.
// If this reports true then v.Interface() == x will not panic for any x,
// nor will v.Equal(u) for any Value u.
func (v Value) Comparable() bool {
	k := v.Kind()
	switch k {
	case Invalid:
		return false

	case Array:
		switch v.Type().Elem().Kind() {
		case Interface, Array, Struct:
			for i := 0; i < v.Type().Len(); i++ {
				if !v.Index(i).Comparable() {
					return false
				}
			}
			return true
		}
		return v.Type().Comparable()

	case Interface:
		return v.IsNil() || v.Elem().Comparable()

	case Struct:
		for i := 0; i < v.NumField(); i++ {
			if !v.Field(i).Comparable() {
				return false
			}
		}
		return true

	default:
		return v.Type().Comparable()
	}
}

// Clear clears the contents of a map or zeros the contents of a slice.
//
// It panics if v's Kind is not [Map] or [Slice].
func (v Value) Clear() {
	switch v.Kind() {
	case Slice:
		sh := *(*unsafeheaderSlice)(v.ptr)
		if sh.Len == 0 || sh.Data == nil {
			return
		}
		st := (*sliceType)(unsafe.Pointer(v.typ()))
		elem := st.Elem
		step := elem.Size()
		for i := 0; i < sh.Len; i++ {
			typedmemclr(elem, unsafe.Add(sh.Data, uintptr(i)*step))
		}
	case Map:
		mapclear(v.typ(), v.pointer())
	default:
		panic(&ValueError{"reflect.Value.Clear", v.Kind()})
	}
}

//go:linkname sliceclear github.com/goplus/llgo/runtime/internal/runtime.SliceClear
func sliceclear(t *abi.Type, s unsafeheaderSlice)

//go:linkname ifaceE2I github.com/goplus/llgo/runtime/internal/runtime.IfaceE2I
func ifaceE2I(t *abi.Type, src any, dst unsafe.Pointer)
