//go:build cgo && !wasm

package cgo

import (
	"runtime"
	"testing"
	"unsafe"
)

func TestCgoImmediateCapture(t *testing.T) {
	p := allocationCreate()
	if p == nil {
		t.Fatal("C allocation failed")
	}
	defer allocationFree(p)
	for i := uint64(0); i < 128; i++ {
		*(*uint64)(p) = i
		if allocationPointer(p) != i || allocationAddress(p) != i || allocationScalar(i) != i+1 {
			t.Fatal("C scalar/pointer result")
		}
		a, b := allocationStruct(i)
		if a != i+1 || b != i&1 {
			t.Fatal("C struct result")
		}
	}
	// Ordinary Go buffers must remain live across the checking closure too.
	value := new(uint64)
	*value = 42
	runtime.GC()
	if allocationPointer(unsafe.Pointer(value)) != 42 {
		t.Fatal("Go buffer lost")
	}
	runtime.KeepAlive(value)
}

func BenchmarkCgoImmediateCapture(b *testing.B) {
	p := allocationCreate()
	if p == nil {
		b.Fatal("C allocation failed")
	}
	defer allocationFree(p)
	*(*uint64)(p) = 17
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if allocationPointer(p) != 17 {
			b.Fatal("incorrect result")
		}
	}
}
