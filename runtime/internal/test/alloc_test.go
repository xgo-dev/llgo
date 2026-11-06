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
			{"AllocNoScanU", rt.AllocNoScanU},
			{"AllocNoScanZ", rt.AllocNoScanZ},
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
				if (alloc.name == "AllocZ" || alloc.name == "AllocNoScanZ") && data[i] != 0 {
					t.Fatalf("%s(%d) byte %d = %d", alloc.name, size, i, data[i])
				}
				data[i] = 0xa5
			}
			if alloc.name == "AllocRoot" {
				rt.FreeRoot(ptr)
			}
		}
	}
}

func TestZeroSizeAllocatorsShareBase(t *testing.T) {
	base := rt.AllocU(0)
	base = atomic.LoadPointer(&base)
	if base == nil {
		t.Fatal("zero-sized allocation returned nil")
	}
	var zero uintptr
	for i := 0; i < 100; i++ {
		// Keep the size dynamic so calls exercise the runtime zero-size path.
		size := atomic.LoadUintptr(&zero)
		for _, alloc := range []struct {
			name string
			fn   func(uintptr) unsafe.Pointer
		}{
			{"AllocU", rt.AllocU},
			{"AllocNoScanU", rt.AllocNoScanU},
			{"AllocNoScanZ", rt.AllocNoScanZ},
			{"AllocZ", rt.AllocZ},
			{"AllocRoot", rt.AllocRoot},
		} {
			ptr := alloc.fn(size)
			ptr = atomic.LoadPointer(&ptr)
			if ptr != base {
				t.Fatalf("%s(0) = %p, want shared base %p", alloc.name, ptr, base)
			}
			if alloc.name == "AllocRoot" {
				// Repeated frees of the static base must never reach a heap allocator.
				rt.FreeRoot(ptr)
			}
		}
	}
	ptr := rt.AllocRoot(16)
	if ptr == base {
		t.Fatal("nonzero allocation returned zerobase")
	}
	*(*uintptr)(ptr) = 42
	rt.FreeRoot(ptr)
}
