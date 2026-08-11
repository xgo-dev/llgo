package emscripten

import (
	"testing"
	"unsafe"
)

func TestFiberStorageUsesEightWords(t *testing.T) {
	if got, want := unsafe.Sizeof(Fiber{}), uintptr(8)*unsafe.Sizeof(uintptr(0)); got != want {
		t.Fatalf("Fiber size = %d, want %d", got, want)
	}
}
