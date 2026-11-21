//go:build !llgo || !wasm || (!wasip1 && (!js || llgo.wasm.reflect.libffi))

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
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

package reflect

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llgo/runtime/internal/ffi"
	"github.com/xgo-dev/llgo/runtime/internal/runtime"
)

func resetWasmFuncBridge(ft *abi.FuncType) {}

func copyWasmFuncBridge(dst, src *abi.FuncType) {}

var ffiTypeClosure = ffi.StructOf(ffi.TypePointer, ffi.TypePointer)

func toFFIType(typ *abi.Type) *ffi.Type {
	// Vector storage (including zero-length arrays and enclosing aggregates)
	// has alignment above the ordinary Go maximum of 8. libffi's scalar/struct
	// descriptions cannot represent its vector calling convention. Check before
	// skipping zero-sized fields; pointers and slice headers retain scalar ABI.
	if typ.Align_ > 8 {
		panic("reflect: libffi calls with vector storage are not supported")
	}
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
		if typ.IsClosure() {
			return ffiTypeClosure
		}
		return ffiWidenGoPointerStruct(toFFIStructType(typ), typ)
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
	// Ordinary trailing zero-sized fields can enlarge the Go-visible size
	// without changing llgo's callable ABI. HostLayout keeps that tail
	// protection in LLVM storage, including in enclosing aggregates, so
	// the FFI description must match Size_ wherever that padding is also
	// part of the calling convention.
	if off < typ.Size_ && structHasHostLayout(typ) && ffiHostLayoutTailPadding {
		fields, off = appendFFIPadding(fields, off, typ.Size_-off)
	}
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

func omitFFIArg(typ *abi.Type) bool {
	// Keep the vector-storage diagnostic in toFFIType, including for empty
	// arrays with vector alignment.
	return ffiSkipEmptyArgs && typ.Size_ == 0 && typ.Align_ <= 8
}

func toFFISig(tin, tout []*abi.Type) (*ffi.Signature, error) {
	args := make([]*ffi.Type, 0, len(tin))
	for _, in := range tin {
		if omitFFIArg(in) {
			continue
		}
		args = append(args, toFFIType(in))
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

func callProvider(ft *abi.FuncType, fn, env unsafe.Pointer, receiverType *abi.Type, receiver *unsafe.Pointer, in []Value) (out []Value) {
	ffiArgs := make([]*ffi.Type, 0, len(in)+4)
	var args []unsafe.Pointer
	if receiver != nil {
		ffiArgs = append(ffiArgs, toFFIType(receiverType))
		args = append(args, unsafe.Pointer(receiver))
	} else if env != nil && ffi.ClosureEnvExplicit {
		ffiArgs = append(ffiArgs, ffi.TypePointer)
		args = append(args, unsafe.Pointer(&env))
	}
	for i, arg := range in {
		typ := ft.In[i]
		if omitFFIArg(typ) {
			continue
		}
		if ffiCallSliceAsTriple && typ.Kind() == abi.Slice {
			h := (*unsafeheaderSlice)(arg.ptr)
			ffiArgs = append(ffiArgs, ffi.TypePointer, ffi.TypeInt, ffi.TypeInt)
			args = append(args, unsafe.Pointer(&h.Data), unsafe.Pointer(&h.Len), unsafe.Pointer(&h.Cap))
			continue
		}
		ffiArgs = append(ffiArgs, toFFIType(typ))
		args = append(args, toFFIArg(arg, typ))
	}

	sig, err := ffi.NewSignature(toFFIRetType(ft.Out), ffiArgs...)
	if err != nil {
		panic(err)
	}
	var ret unsafe.Pointer
	if sig.ReturnType() != ffi.TypeVoid {
		// libffi expects rvalue to point to the return value storage.
		ret = runtime.AllocZAligned(sig.ReturnType().ByteSize(), uintptr(sig.ReturnType().Alignment))
	}

	ffi.CallWithEnv(sig, fn, env, ret, args...)
	tout := toRuntimeTypes(ft.Out)
	switch n := len(tout); n {
	case 0:
	case 1:
		return []Value{valueFromFFIResult(ret, tout[0], sig.ReturnType())}
	default:
		out = make([]Value, n)
		var off uintptr
		for i, tout := range tout {
			field, fieldOffset, next := ffiResultField(sig.ReturnType(), i, off)
			out[i] = valueFromFFIResult(add(ret, fieldOffset, ""), tout, field)
			off = next
		}
	}
	return
}

func valueFromFFIResult(ptr unsafe.Pointer, typ *abi.Type, ffiType *ffi.Type) Value {
	if ffiType.ByteSize() != typ.Size_ || ffiType.Alignment < uint16(typ.Align_) {
		storage := runtime.AllocZAligned(typ.Size_, uintptr(typ.Align_))
		size := ffiType.ByteSize()
		if typ.Size_ < size {
			size = typ.Size_
		}
		if size != 0 {
			memmove(storage, ptr, size)
		}
		ptr = storage
	}
	out := NewAt(toType(typ), ptr).Elem()
	resolveIndirectValue(&out, typ)
	return out
}
