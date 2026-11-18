package emscripten

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestFiberStorageUsesEightNativeWords(t *testing.T) {
	if got, want := unsafe.Sizeof(Fiber{}), expectedFiberSize; got != want {
		t.Fatalf("Fiber size = %d, want %d", got, want)
	}
	var fibers [2]Fiber
	stride := uintptr(unsafe.Pointer(&fibers[1])) - uintptr(unsafe.Pointer(&fibers[0]))
	if stride != expectedFiberSize {
		t.Fatalf("Fiber storage stride = %d, want %d", stride, expectedFiberSize)
	}
}

func TestFiberHasNoReflectableHostMethods(t *testing.T) {
	if got := reflect.TypeOf(Fiber{}).NumMethod(); got != 0 {
		t.Fatalf("Fiber has %d reflectable methods, want 0", got)
	}
}
