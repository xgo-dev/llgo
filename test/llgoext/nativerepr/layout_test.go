package nativerepr

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/test/llgoext/nativerepr/derived"
	"github.com/xgo-dev/llgo/test/llgoext/nativerepr/native"
)

//llgo:cdecl
type nativeEmptyTail struct {
	X float64
	Z struct{}
}

type derivedEmptyTail nativeEmptyTail

//llgo:cdecl
type nativeAlias = struct {
	B    byte
	Fn   func()
	Tail byte
}

type aliasCopy nativeAlias

//llgo:cdecl
type nativeCallbackArray [3]nativeAbsCallback

type callbackArrayCopy nativeCallbackArray

//llgo:cdecl
type nativeUintptrArray [3]uintptr

type uintptrArrayCopy nativeUintptrArray

type nativeContainer struct {
	Prefix byte
	Values [3]localCopy
	Next   *localCopy
	Z      struct{}
}

func checkNativeLayout(t *testing.T, value any, size, alignment uintptr, offsets ...uintptr) {
	t.Helper()
	rt := reflect.TypeOf(value)
	if rt.Size() != size || uintptr(rt.Align()) != alignment {
		t.Fatalf("%s reflection size/align = %d/%d; unsafe = %d/%d", rt, rt.Size(), rt.Align(), size, alignment)
	}
	for i, offset := range offsets {
		if got := rt.Field(i).Offset; got != offset {
			t.Fatalf("%s field %d reflection offset = %d; unsafe = %d", rt, i, got, offset)
		}
	}
}

func TestNativeInheritedSourceAndReflectionLayout(t *testing.T) {
	var root native.Record
	var copy localCopy
	checkNativeLayout(t, root, unsafe.Sizeof(root), unsafe.Alignof(root), unsafe.Offsetof(root.Fn), unsafe.Offsetof(root.Tail))
	checkNativeLayout(t, copy, unsafe.Sizeof(copy), unsafe.Alignof(copy), unsafe.Offsetof(copy.Fn), unsafe.Offsetof(copy.Tail))
	checkNativeLayout(t, derived.Copy{}, unsafe.Sizeof(derived.Copy{}), unsafe.Alignof(derived.Copy{}), 0, unsafe.Offsetof(derived.Copy{}.Tail))
	if unsafe.Sizeof(root) != unsafe.Sizeof(copy) || unsafe.Offsetof(root.Tail) != unsafe.Offsetof(copy.Tail) {
		t.Fatal("defined type changed inherited layout")
	}
	if offset := uintptr(unsafe.Pointer(&copy.Tail)) - uintptr(unsafe.Pointer(&copy)); offset != unsafe.Offsetof(copy.Tail) {
		t.Fatalf("actual Tail offset = %d; source = %d", offset, unsafe.Offsetof(copy.Tail))
	}
	var values [3]localCopy
	if stride := uintptr(unsafe.Pointer(&values[1])) - uintptr(unsafe.Pointer(&values[0])); stride != unsafe.Sizeof(copy) {
		t.Fatalf("actual array stride = %d; source size = %d", stride, unsafe.Sizeof(copy))
	}
	if rt := reflect.TypeOf(values); rt.Size() != unsafe.Sizeof(values) || rt.Elem().Size() != unsafe.Sizeof(copy) {
		t.Fatal("array metadata lost inherited element size")
	}
	var container nativeContainer
	checkNativeLayout(t, container, unsafe.Sizeof(container), unsafe.Alignof(container), 0, unsafe.Offsetof(container.Values), unsafe.Offsetof(container.Next), unsafe.Offsetof(container.Z))
	checkNativeLayout(t, nativeEmptyTail{}, unsafe.Sizeof(nativeEmptyTail{}), unsafe.Alignof(nativeEmptyTail{}), 0, unsafe.Offsetof(nativeEmptyTail{}.Z))
	checkNativeLayout(t, derivedEmptyTail{}, unsafe.Sizeof(derivedEmptyTail{}), unsafe.Alignof(derivedEmptyTail{}), 0, unsafe.Offsetof(derivedEmptyTail{}.Z))
	checkNativeLayout(t, nativeAlias{}, unsafe.Sizeof(nativeAlias{}), unsafe.Alignof(nativeAlias{}), 0, unsafe.Offsetof(nativeAlias{}.Fn), unsafe.Offsetof(nativeAlias{}.Tail))
	checkNativeLayout(t, aliasCopy{}, unsafe.Sizeof(aliasCopy{}), unsafe.Alignof(aliasCopy{}), 0, unsafe.Offsetof(aliasCopy{}.Fn), unsafe.Offsetof(aliasCopy{}.Tail))
}

func TestNativeInheritedArrayStorage(t *testing.T) {
	callbacks := nativeCallbackArray{nativeAbsCallback(nativeAbs), nil, nativeAbsCallback(nativeAbs)}
	copy := (*callbackArrayCopy)(&callbacks)
	copy[1] = copy[0]
	for i := range callbacks {
		if got := callbacks[i](-23); got != 23 {
			t.Fatalf("callback %d returned %d", i, got)
		}
	}
	if stride := uintptr(unsafe.Pointer(&callbacks[1])) - uintptr(unsafe.Pointer(&callbacks[0])); 3*stride != unsafe.Sizeof(callbacks) {
		t.Fatalf("callback array stride/size = %d/%d", stride, unsafe.Sizeof(callbacks))
	}
	if got := reflect.TypeOf(callbacks).Size(); got != unsafe.Sizeof(callbacks) {
		t.Fatalf("callback array reflection/source size = %d/%d", got, unsafe.Sizeof(callbacks))
	}
	values := nativeUintptrArray{11, 22, 33}
	ints := (*uintptrArrayCopy)(&values)
	ints[1] = 44
	if values != (nativeUintptrArray{11, 44, 33}) {
		t.Fatalf("native uintptr indexing changed adjacent elements: %v", values)
	}
	if stride := uintptr(unsafe.Pointer(&values[1])) - uintptr(unsafe.Pointer(&values[0])); 3*stride != unsafe.Sizeof(values) {
		t.Fatalf("uintptr array stride/size = %d/%d", stride, unsafe.Sizeof(values))
	}
	if got := reflect.TypeOf(values).Size(); got != unsafe.Sizeof(values) {
		t.Fatalf("uintptr array reflection/source size = %d/%d", got, unsafe.Sizeof(values))
	}
}
