// LITTEST
// Scope: common
// CHECK: define {{.*}} @main.main(
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

package main

import "unsafe"

type GoRecord[T any] struct {
	Ptr  *byte
	Tail T
}

//llgo:cdecl
type NativeFromGo[T any] GoRecord[T]

//llgo:cdecl
type NativeRecord[T any] struct {
	Ptr  *byte
	Tail T
}

type GoFromNative[T any] NativeRecord[T]

type NativeWrapper[T any] struct {
	Pad byte
	NativeFromGo[T]
}

type GoWrapper[T any] struct {
	Pad byte
	GoFromNative[T]
}

func layout[T any](v T) (uintptr, uintptr) {
	return unsafe.Sizeof(v), unsafe.Alignof(v)
}

func nativeOffset[T any](v *NativeFromGo[T]) uintptr    { return unsafe.Offsetof(v.Tail) }
func goOffset[T any](v *GoFromNative[T]) uintptr        { return unsafe.Offsetof(v.Tail) }
func nativePromoted[T any](v *NativeWrapper[T]) uintptr { return unsafe.Offsetof(v.Tail) }
func goPromoted[T any](v *GoWrapper[T]) uintptr         { return unsafe.Offsetof(v.Tail) }

func checkLayout[T any](values *[2]T, actualAlign uintptr) {
	size, alignment := layout(values[0])
	stride := uintptr(unsafe.Pointer(&values[1])) - uintptr(unsafe.Pointer(&values[0]))

	if size != stride || alignment != actualAlign {
		println("layout", size, alignment, "storage", stride, actualAlign)
		panic("named layout differs from storage")
	}
	arraySize, _ := layout(*values)
	if arraySize != 2*stride {
		panic("array layout differs from storage")
	}
}

func main() {
	var native [2]NativeFromGo[uint32]
	// The pointer is the largest-aligned field; Tail starts at that alignment.
	checkLayout(&native, uintptr(unsafe.Pointer(&native[0].Tail))-uintptr(unsafe.Pointer(&native[0])))
	if nativeOffset(&native[0]) != uintptr(unsafe.Pointer(&native[0].Tail))-uintptr(unsafe.Pointer(&native[0])) {
		panic("native field offset differs from storage")
	}
	var goValues [2]GoFromNative[uint32]
	checkLayout(&goValues, uintptr(unsafe.Pointer(&goValues[0].Tail))-uintptr(unsafe.Pointer(&goValues[0])))
	if goOffset(&goValues[0]) != uintptr(unsafe.Pointer(&goValues[0].Tail))-uintptr(unsafe.Pointer(&goValues[0])) {
		panic("Go field offset differs from storage")
	}
	var n NativeWrapper[uint32]
	if nativePromoted(&n) != uintptr(unsafe.Pointer(&n.Tail))-uintptr(unsafe.Pointer(&n)) {
		panic("native promoted offset differs from storage")
	}
	var g GoWrapper[uint32]
	if goPromoted(&g) != uintptr(unsafe.Pointer(&g.Tail))-uintptr(unsafe.Pointer(&g)) {
		panic("Go promoted offset differs from storage")
	}
}

//llgo:cdecl
type CallbackRecord[T any] struct {
	Fn   func()
	Tail T
}

//llgo:cdecl
type CCallback func()

type CallbackControls[T any] struct {
	C    CCallback
	Std  CCallback
	Tail T
}
type CallbackWrapper[T any] struct {
	Pad byte
	CallbackRecord[T]
}
type ZeroEnd[T any] struct {
	Value T
	End   [0]byte
}

func callbackOffset[T any](v *CallbackRecord[T]) uintptr    { return unsafe.Offsetof(v.Tail) }
func controlOffset[T any](v *CallbackControls[T]) uintptr   { return unsafe.Offsetof(v.Tail) }
func callbackPromoted[T any](v *CallbackWrapper[T]) uintptr { return unsafe.Offsetof(v.Tail) }
func callbackExplicit[T any](v *CallbackWrapper[T]) uintptr {
	return unsafe.Offsetof(v.CallbackRecord.Tail)
}

func init() {
	var callbacks [2]CallbackRecord[int32]
	size, _ := layout(callbacks[0])
	if size != uintptr(unsafe.Pointer(&callbacks[1]))-uintptr(unsafe.Pointer(&callbacks[0])) {
		panic("callback record size differs from stride")
	}
	if callbackOffset(&callbacks[0]) != uintptr(unsafe.Pointer(&callbacks[0].Tail))-uintptr(unsafe.Pointer(&callbacks[0])) {
		panic("callback offset differs from storage")
	}
	var controls CallbackControls[int32]
	if controlOffset(&controls) != uintptr(unsafe.Pointer(&controls.Tail))-uintptr(unsafe.Pointer(&controls)) {
		panic("named callback offset differs from storage")
	}
	var wrapper CallbackWrapper[int32]
	if callbackPromoted(&wrapper) != uintptr(unsafe.Pointer(&wrapper.Tail))-uintptr(unsafe.Pointer(&wrapper)) {
		panic("callback promoted offset differs from storage")
	}
	if callbackExplicit(&wrapper) != uintptr(unsafe.Pointer(&wrapper.Tail))-uintptr(unsafe.Pointer(&wrapper.CallbackRecord)) {
		panic("callback explicit offset differs from storage")
	}
	var zeros [2]ZeroEnd[uint64]
	size, _ = layout(zeros[0])
	if size != uintptr(unsafe.Pointer(&zeros[1]))-uintptr(unsafe.Pointer(&zeros[0])) {
		panic("zero end size differs from stride")
	}
	arraySize, _ := layout(zeros)
	if arraySize != 2*size {
		panic("zero end array size differs from stride")
	}
}

type EmbeddedZero[T any] struct {
	ZeroEnd[T]
	Tail uint64
}

func embeddedZeroOffset[T any](p *EmbeddedZero[T]) uintptr { return unsafe.Offsetof(p.Tail) }
func init() {
	var values [2]EmbeddedZero[uint64]
	actual := uintptr(unsafe.Pointer(&values[0].Tail)) - uintptr(unsafe.Pointer(&values[0]))
	if embeddedZeroOffset(&values[0]) != actual {
		panic("embedded zero-tail offset differs from storage")
	}
	size, _ := layout(values[0])
	if size != uintptr(unsafe.Pointer(&values[1]))-uintptr(unsafe.Pointer(&values[0])) {
		panic("embedded zero-tail size differs from stride")
	}
}

//llgo:cdecl
type AlignedNative[T any] struct {
	Pad   byte
	Value T
}
type AlignedOuter[T any] struct {
	Prefix byte
	AlignedNative[T]
	Tail byte
}
type AlignedArrayOuter[T any] struct {
	Prefix byte
	Values [2]AlignedNative[T]
	Tail   byte
}

func outerOffsets[T any](p *AlignedOuter[T]) (uintptr, uintptr, uintptr) {
	return unsafe.Offsetof(p.AlignedNative), unsafe.Offsetof(p.Value), unsafe.Offsetof(p.Tail)
}
func arrayOffsets[T any](p *AlignedArrayOuter[T]) (uintptr, uintptr) {
	return unsafe.Offsetof(p.Values), unsafe.Offsetof(p.Tail)
}
func init() {
	var o AlignedOuter[uint64]
	base := uintptr(unsafe.Pointer(&o))
	embedded, value, tail := outerOffsets(&o)
	if embedded != uintptr(unsafe.Pointer(&o.AlignedNative))-base || value != uintptr(unsafe.Pointer(&o.Value))-base || tail != uintptr(unsafe.Pointer(&o.Tail))-base {
		panic("native-containing offsets differ from storage")
	}
	var a AlignedArrayOuter[uint64]
	field, end := arrayOffsets(&a)
	base = uintptr(unsafe.Pointer(&a))
	if field != uintptr(unsafe.Pointer(&a.Values))-base || end != uintptr(unsafe.Pointer(&a.Tail))-base {
		panic("native-array-containing offsets differ from storage")
	}
}

//llgo:cdecl
type NativeValue[T any] struct {
	Value T
	Tail  byte
}
type GoValue[T any] struct {
	Value T
	Tail  byte
}

func nativeFieldSize[T any](p *NativeValue[T]) uintptr { return unsafe.Sizeof(p.Value) }
func nativeValueSize[T any](p NativeValue[T]) uintptr  { return unsafe.Sizeof(p.Value) }
func goFieldSize[T any](p *GoValue[T]) uintptr         { return unsafe.Sizeof(p.Value) }
func nativeEffect[T any]() *NativeValue[T]             { panic("native field operand evaluated") }
func nativeEffectSize[T any]() uintptr                 { return unsafe.Sizeof(nativeEffect[T]().Value) }
func init() {
	var n NativeValue[func()]
	// Layout queries use loaded values; Memory32 callback values can be
	// wider than their physical native slots.
	actual, _ := layout(CCallback(nil))
	if nativeFieldSize(&n) != actual || nativeValueSize(n) != actual || nativeEffectSize[func()]() != actual {
		panic("native callback field value size mismatch")
	}
	var a NativeValue[[2]func()]
	if nativeFieldSize(&a) != 2*actual || nativeValueSize(a) != 2*actual {
		panic("native callback array field value size mismatch")
	}
	var g GoValue[func()]
	if goFieldSize(&g) != uintptr(unsafe.Pointer(&g.Tail))-uintptr(unsafe.Pointer(&g)) {
		panic("Go callback field size differs from storage")
	}
	var c NativeValue[CCallback]
	if nativeFieldSize(&c) != actual {
		panic("typed C callback field size differs from storage")
	}
}

func nativeIndexSize[T any](p *NativeValue[[2]T]) uintptr {
	return unsafe.Sizeof(p.Value[layoutIndex()])
}
func layoutIndex() int                                  { panic("layout index evaluated") }
func nativeDerefSize[T any](p *NativeValue[T]) uintptr  { return unsafe.Sizeof(*(&p.Value)) }
func nativeFieldAlign[T any](p *NativeValue[T]) uintptr { return unsafe.Alignof(p.Value) }
func init() {
	var a NativeValue[[2]func()]
	callbackSize, _ := layout(CCallback(nil))
	if nativeIndexSize(&a) != callbackSize {
		panic("native indexed callback size mismatch")
	}
	var n NativeValue[func()]
	if nativeDerefSize(&n) != callbackSize {
		panic("native dereferenced callback size mismatch")
	}
	var u NativeValue[uintptr]
	size, align := layout(uintptr(0))
	if nativeFieldSize(&u) != size || nativeFieldAlign(&u) != align || nativeDerefSize(&u) != size {
		panic("native uintptr value layout mismatch")
	}
}
