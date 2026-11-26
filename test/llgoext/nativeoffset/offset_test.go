package nativeoffset

import (
	"testing"
	"unsafe"

	native "github.com/xgo-dev/llgo/test/llgoext/nativeoffset/offsetdep"
)

type offsetGo struct {
	Head byte
	Fn   func()
	Tail uintptr
}

//llgo:cdecl
type offsetNative offsetGo
type offsetAlias = offsetNative
type offsetOuter struct {
	Head byte
	offsetAlias
}

const nativeOffset = unsafe.Offsetof(offsetNative{}.Tail)
const aliasOffset = unsafe.Offsetof(offsetAlias{}.Tail)
const goOffset = unsafe.Offsetof(offsetGo{}.Tail)
const crossPackageOffset = unsafe.Offsetof(native.Record{}.Tail)
const (
	repeatedOffset = unsafe.Offsetof(offsetOuter{}.Tail) + iota
	repeatedOffset1
	repeatedOffset2
)

// The checker must produce the right constant before constructing this type.
type offsetBuffer [aliasOffset]byte

//llgo:noinline
func variableOffset[T any](v struct {
	Head T
	Tail byte
}) uintptr {
	return unsafe.Offsetof(v.Tail)
}

func TestNativeOffsetConstants(t *testing.T) {
	var n offsetNative
	var d offsetAlias
	var g offsetGo
	var c native.Record
	var outer offsetOuter
	for _, check := range []struct {
		name              string
		constant, address uintptr
	}{
		{"native", nativeOffset, uintptr(unsafe.Pointer(&n.Tail)) - uintptr(unsafe.Pointer(&n))},
		{"alias", aliasOffset, uintptr(unsafe.Pointer(&d.Tail)) - uintptr(unsafe.Pointer(&d))},
		{"go", goOffset, uintptr(unsafe.Pointer(&g.Tail)) - uintptr(unsafe.Pointer(&g))},
		{"cross-package", crossPackageOffset, uintptr(unsafe.Pointer(&c.Tail)) - uintptr(unsafe.Pointer(&c))},
		{"array-bound", uintptr(len(offsetBuffer{})), uintptr(unsafe.Pointer(&d.Tail)) - uintptr(unsafe.Pointer(&d))},
		{"promoted", repeatedOffset, uintptr(unsafe.Pointer(&outer.Tail)) - uintptr(unsafe.Pointer(&outer))},
		{"repeated-const", repeatedOffset2 - 2, uintptr(unsafe.Pointer(&outer.Tail)) - uintptr(unsafe.Pointer(&outer))},
	} {
		if check.constant != check.address {
			t.Errorf("%s: constant %d, address %d", check.name, check.constant, check.address)
		}
	}
	type Generic = struct {
		Head native.Record
		Tail byte
	}
	var v Generic
	if got, want := variableOffset(v), uintptr(unsafe.Pointer(&v.Tail))-uintptr(unsafe.Pointer(&v)); got != want {
		t.Fatalf("generic offset %d, address %d", got, want)
	}
}
