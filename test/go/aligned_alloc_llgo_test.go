//go:build llgo && (386 || amd64 || arm64 || wasm)

package gotest

import (
	"runtime"
	"testing"
	"unsafe"
)

//go:linkname alignedAllocation github.com/xgo-dev/llgo/runtime/internal/runtime.AllocZAligned
func alignedAllocation(size, alignment uintptr) unsafe.Pointer

var alignedRoots []unsafe.Pointer

func TestRuntimeAlignedAllocationGC(t *testing.T) {
	// Exercise 16-byte through 16-KiB alignment, including requests beyond
	// GC_memalign's native limit and roots inside collector-owned blocks.
	alignedRoots = nil
	for i := 0; i < 160; i++ {
		alignment := uintptr(16) << uint(i%11)
		p := alignedAllocation(37, alignment)
		if p == nil || uintptr(p)&(alignment-1) != 0 {
			t.Fatal("aligned allocation", alignment)
		}
		data := unsafe.Slice((*byte)(p), 37)
		for j, b := range data {
			if b != 0 {
				t.Fatal("nonzero allocation")
			}
			data[j] = byte(i + j)
		}
		alignedRoots = append(alignedRoots, p)
	}
	runtime.GC()
	for i, p := range alignedRoots {
		for j, b := range unsafe.Slice((*byte)(p), 37) {
			if b != byte(i+j) {
				t.Fatal("aligned allocation did not survive GC", i, j)
			}
		}
	}
	if p := alignedAllocation(0, 64); p == nil || uintptr(p)&63 != 0 {
		t.Fatal("aligned zero-sized allocation")
	}
	for _, bad := range [][2]uintptr{
		{1, 0}, {1, 3}, {^uintptr(0), 16}, {^uintptr(0), 1},
		{1, uintptr(1) << (unsafe.Sizeof(uintptr(0))*8 - 1)},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("invalid aligned allocation accepted", bad)
				}
			}()
			alignedAllocation(bad[0], bad[1])
		}()
	}
}

// Run this with nogc as well: GC_malloc already returns zeroed memory and can
// hide a missing clear in AllocZAligned. Linux CI poisons malloc allocations
// with MALLOC_PERTURB_ so the nogc case does not rely on allocator reuse.
func TestRuntimeAlignedAllocationZeroed(t *testing.T) {
	for _, alignment := range []uintptr{1, unsafe.Sizeof(uintptr(0)), 16, 32, 64, 4096, 16384} {
		for _, size := range []uintptr{0, 1, 37, 4093} {
			p := alignedAllocation(size, alignment)
			if p == nil || uintptr(p)&(alignment-1) != 0 {
				t.Fatalf("size %d alignment %d: misaligned allocation", size, alignment)
			}
			for i, b := range unsafe.Slice((*byte)(p), int(size)) {
				if b != 0 {
					t.Fatalf("size %d alignment %d: byte %d = %#x, want zero", size, alignment, i, b)
				}
			}
		}
	}
}
