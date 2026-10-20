//go:build llgo && cgo && (amd64 || arm64)

package cinterop

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestSIMDCInteropImportedLayout(t *testing.T) {
	var v cFloat4
	var b cBox
	if unsafe.Sizeof(v) != 16 || unsafe.Alignof(v) != 16 {
		t.Fatalf("vector size/align = %d/%d", unsafe.Sizeof(v), unsafe.Alignof(v))
	}
	layout := unsafe.Sizeof(b)*10000 + unsafe.Offsetof(b.v)*100 + unsafe.Alignof(b)
	if uint64(layout) != nativeBoxLayout() {
		t.Fatalf("Go/C layout = %d/%d", layout, nativeBoxLayout())
	}
	if got := uintptr(unsafe.Pointer(&b.v)) - uintptr(unsafe.Pointer(&b)); got != unsafe.Offsetof(b.v) {
		t.Fatalf("pointer offset %d, constant %d", got, unsafe.Offsetof(b.v))
	}
	var plain ordinaryArray
	if unsafe.Sizeof(plain) != 16 || unsafe.Alignof(plain) != 4 {
		t.Fatal("ordinary Go array became a vector")
	}
	for _, value := range []any{v, b, [2]cFloat4{}} {
		rt := reflect.TypeOf(value)
		if rt.Align() != 16 {
			t.Fatalf("reflect alignment for %v = %d", rt, rt.Align())
		}
	}
	field, ok := reflect.TypeOf(b).FieldByName("v")
	if !ok || reflect.TypeOf(b).Size() != unsafe.Sizeof(b) || field.Offset != unsafe.Offsetof(b.v) {
		t.Fatal("reflect struct layout")
	}
}
