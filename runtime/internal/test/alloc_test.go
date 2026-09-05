//go:build llgo

package test

import (
	"sync/atomic"
	"testing"
	"unsafe"

	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

func TestAllocatorNonNull(t *testing.T) {
	for _, size := range []uintptr{0, 1, 64} {
		for _, alloc := range []struct {
			name string
			fn   func(uintptr) unsafe.Pointer
		}{
			{"AllocU", rt.AllocU},
			{"AllocZ", rt.AllocZ},
			{"AllocRoot", rt.AllocRoot},
		} {
			ptr := alloc.fn(size)
			// Reload from memory so the compiler cannot discard the assertion
			// based on the allocator's nonnull return attribute.
			ptr = atomic.LoadPointer(&ptr)
			if ptr == nil {
				t.Fatalf("%s(%d) returned nil", alloc.name, size)
			}
			data := unsafe.Slice((*byte)(ptr), size)
			for i := range data {
				if alloc.name == "AllocZ" && data[i] != 0 {
					t.Fatalf("AllocZ(%d) byte %d = %d", size, i, data[i])
				}
				data[i] = 0xa5
			}
			if alloc.name == "AllocRoot" {
				rt.FreeRoot(ptr)
			}
		}
	}
}
