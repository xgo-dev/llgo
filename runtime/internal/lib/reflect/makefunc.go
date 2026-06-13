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

// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// MakeFunc implementation.

package reflect

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/ffi"
)

type funcData struct {
	ftyp *funcType
	fn   func(args []Value) (results []Value)
	nin  int
	off  int
}

func MakeFunc(typ Type, fn func(args []Value) (results []Value)) Value {
	return makeFunc(typ, false, fn)
}

func makeFunc(typ Type, method bool, fn func(args []Value) (results []Value)) Value {
	if typ.Kind() != Func {
		panic("reflect: call of MakeFunc with non-Func type")
	}

	t := typ.common()
	ftyp := (*funcType)(unsafe.Pointer(t))
	var ins []*abi.Type
	var off int
	if method {
		ins = ftyp.In
	} else {
		ins = append([]*abi.Type{unsafePointerType}, ftyp.In...)
		off = 1
	}
	sig, err := toFFISig(ins, ftyp.Out)
	if err != nil {
		panic(err)
	}
	closure := ffi.NewClosure()

	switch len(ftyp.Out) {
	case 0:
		err = closure.Bind(sig, func(cif *ffi.Signature, ret unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer) {
			fd := (*funcData)(userdata)
			ins := make([]Value, fd.nin)
			for i := 0; i < fd.nin; i++ {
				ins[i] = ffiToValue(ffi.Index(args, uintptr(i+fd.off)), fd.ftyp.In[i])
			}
			validateMakeFuncResults(fd.fn(ins), fd.ftyp)
		}, unsafe.Pointer(&funcData{ftyp: ftyp, fn: fn, nin: len(ftyp.In), off: off}))
	case 1:
		err = closure.Bind(sig, func(cif *ffi.Signature, ret unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer) {
			fd := (*funcData)(userdata)
			ins := make([]Value, fd.nin)
			for i := 0; i < fd.nin; i++ {
				ins[i] = ffiToValue(ffi.Index(args, uintptr(i+fd.off)), fd.ftyp.In[i])
			}
			out := validateMakeFuncResults(fd.fn(ins), fd.ftyp)
			storeMakeFuncResult(ret, out[0], fd.ftyp.Out[0])
		}, unsafe.Pointer(&funcData{ftyp: ftyp, fn: fn, nin: len(ftyp.In), off: off}))
	default:
		err = closure.Bind(sig, func(cif *ffi.Signature, ret unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer) {
			fd := (*funcData)(userdata)
			ins := make([]Value, fd.nin)
			for i := 0; i < fd.nin; i++ {
				ins[i] = ffiToValue(ffi.Index(args, uintptr(i+fd.off)), fd.ftyp.In[i])
			}
			outs := validateMakeFuncResults(fd.fn(ins), fd.ftyp)
			var offset uintptr = 0
			alignment := uintptr(cif.RType.Alignment)
			for i, out := range outs {
				typ := fd.ftyp.Out[i]
				storeMakeFuncResult(add(ret, offset, ""), out, typ)
				offset += (typ.Size_ + alignment - 1) &^ (alignment - 1)
			}
		}, unsafe.Pointer(&funcData{ftyp: ftyp, fn: fn, nin: len(ftyp.In), off: off}))
	}
	if err != nil {
		panic("libffi error: " + err.Error())
	}
	styp := closureOf(ftyp)
	fv := &struct {
		fn  unsafe.Pointer
		env unsafe.Pointer
	}{closure.Fn, unsafe.Pointer(&fn)}
	return Value{styp, unsafe.Pointer(fv), flagIndir | flag(Func)}
}

func validateMakeFuncResults(out []Value, ftyp *funcType) []Value {
	if len(out) != len(ftyp.Out) {
		panic("reflect: wrong return count from function created by MakeFunc")
	}
	for i, typ := range ftyp.Out {
		v := out[i]
		if v.typ() == nil {
			panic("reflect: function created by MakeFunc returned zero Value")
		}
		if v.flag&flagRO != 0 {
			panic("reflect: function created by MakeFunc returned value obtained from unexported field")
		}
		out[i] = v.assignTo("reflect: function created by MakeFunc", typ, nil)
	}
	return out
}

func storeMakeFuncResult(ret unsafe.Pointer, v Value, typ *abi.Type) {
	if typ.Size_ == 0 {
		return
	}
	c.Memmove(ret, toFFIArg(v, typ), typ.Size_)
}

func ffiToValue(ptr unsafe.Pointer, typ *abi.Type) (v Value) {
	kind := typ.Kind()
	v.typ_ = typ
	v.flag = flag(kind)
	if typ.IfaceIndir() {
		v.flag |= flagIndir
		v.ptr = ptr
	} else {
		v.ptr = *(*unsafe.Pointer)(ptr)
	}
	return
}

/*
import (
	"unsafe"
)

// makeFuncImpl is the closure value implementing the function
// returned by MakeFunc.
// The first three words of this type must be kept in sync with
// methodValue and runtime.reflectMethodValue.
// Any changes should be reflected in all three.
type makeFuncImpl struct {
	makeFuncCtxt
	ftyp *funcType
	fn   func([]Value) []Value
}

// MakeFunc returns a new function of the given Type
// that wraps the function fn. When called, that new function
// does the following:
//
//   - converts its arguments to a slice of Values.
//   - runs results := fn(args).
//   - returns the results as a slice of Values, one per formal result.
//
// The implementation fn can assume that the argument Value slice
// has the number and type of arguments given by typ.
// If typ describes a variadic function, the final Value is itself
// a slice representing the variadic arguments, as in the
// body of a variadic function. The result Value slice returned by fn
// must have the number and type of results given by typ.
//
// The Value.Call method allows the caller to invoke a typed function
// in terms of Values; in contrast, MakeFunc allows the caller to implement
// a typed function in terms of Values.
//
// The Examples section of the documentation includes an illustration
// of how to use MakeFunc to build a swap function for different types.
func MakeFunc(typ Type, fn func(args []Value) (results []Value)) Value {
	if typ.Kind() != Func {
		panic("reflect: call of MakeFunc with non-Func type")
	}

	t := typ.common()
	ftyp := (*funcType)(unsafe.Pointer(t))

	code := abi.FuncPCABI0(makeFuncStub)

	// makeFuncImpl contains a stack map for use by the runtime
	_, _, abid := funcLayout(ftyp, nil)

	impl := &makeFuncImpl{
		makeFuncCtxt: makeFuncCtxt{
			fn:      code,
			stack:   abid.stackPtrs,
			argLen:  abid.stackCallArgsSize,
			regPtrs: abid.inRegPtrs,
		},
		ftyp: ftyp,
		fn:   fn,
	}

	return Value{t, unsafe.Pointer(impl), flag(Func)}
}

// makeFuncStub is an assembly function that is the code half of
// the function returned from MakeFunc. It expects a *callReflectFunc
// as its context register, and its job is to invoke callReflect(ctxt, frame)
// where ctxt is the context register and frame is a pointer to the first
// word in the passed-in argument frame.
func makeFuncStub()

// The first 3 words of this type must be kept in sync with
// makeFuncImpl and runtime.reflectMethodValue.
// Any changes should be reflected in all three.
type methodValue struct {
	makeFuncCtxt
	method int
	rcvr   Value
}
*/

// makeMethodValue converts v from the rcvr+method index representation
// of a method value to an actual method func value, which is
// basically the receiver value with a special bit set, into a true
// func value - a value holding an actual func. The output is
// semantically equivalent to the input as far as the user of package
// reflect can tell, but the true func representation can be handled
// by code like Convert and Interface and Assign.
func makeMethodValue(op string, v Value) Value {
	if v.flag&flagMethod == 0 {
		panic("reflect: internal error: invalid use of makeMethodValue")
	}

	// Ignoring the flagMethod bit, v describes the receiver, not the method type.
	fl := v.flag & (flagRO | flagAddr | flagIndir)
	fl |= flag(v.typ().Kind())
	rcvr := Value{v.typ(), v.ptr, fl}

	// v.Type returns the actual type of the method value.
	_, _, fn := methodReceiver(op, rcvr, int(v.flag)>>flagMethodShift)
	var ptr unsafe.Pointer
	storeRcvr(v, unsafe.Pointer(&ptr))
	fv := &struct {
		fn  unsafe.Pointer
		env unsafe.Pointer
	}{fn, ptr}
	ftyp := (*funcType)(unsafe.Pointer(v.Type().(*rtype)))
	typ := closureOf(ftyp)
	// Cause panic if method is not appropriate.
	// The panic would still happen during the call if we omit this,
	// but we want Interface() and other operations to fail early.
	return Value{typ, unsafe.Pointer(fv), v.flag&flagRO | flagIndir | flag(Func)}
}

var unsafePointerType = rtypeOf(unsafe.Pointer(nil))

/*
func methodValueCallCodePtr() uintptr {
	return abi.FuncPCABI0(methodValueCall)
}

// methodValueCall is an assembly function that is the code half of
// the function returned from makeMethodValue. It expects a *methodValue
// as its context register, and its job is to invoke callMethod(ctxt, frame)
// where ctxt is the context register and frame is a pointer to the first
// word in the passed-in argument frame.
func methodValueCall()

// This structure must be kept in sync with runtime.reflectMethodValue.
// Any changes should be reflected in all both.
type makeFuncCtxt struct {
	fn      uintptr
	stack   *bitVector // ptrmap for both stack args and results
	argLen  uintptr    // just args
	regPtrs abi.IntArgRegBitmap
}

// moveMakeFuncArgPtrs uses ctxt.regPtrs to copy integer pointer arguments
// in args.Ints to args.Ptrs where the GC can see them.
//
// This is similar to what reflectcallmove does in the runtime, except
// that happens on the return path, whereas this happens on the call path.
//
// nosplit because pointers are being held in uintptr slots in args, so
// having our stack scanned now could lead to accidentally freeing
// memory.
//
//go:nosplit
func moveMakeFuncArgPtrs(ctxt *makeFuncCtxt, args *abi.RegArgs) {
	for i, arg := range args.Ints {
		// Avoid write barriers! Because our write barrier enqueues what
		// was there before, we might enqueue garbage.
		if ctxt.regPtrs.Get(i) {
			*(*uintptr)(unsafe.Pointer(&args.Ptrs[i])) = arg
		} else {
			// We *must* zero this space ourselves because it's defined in
			// assembly code and the GC will scan these pointers. Otherwise,
			// there will be garbage here.
			*(*uintptr)(unsafe.Pointer(&args.Ptrs[i])) = 0
		}
	}
}
*/
