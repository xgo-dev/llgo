//go:build (baremetal && !nogc) || (wasm && llgo.wasm.gc.linear)

package tinygogc

import "unsafe"

// AllocAligned returns zeroed, collector-owned storage, or nil for an invalid
// request. Alignment beyond the heap's guarantee uses an interior pointer.
func AllocAligned(size, alignment uintptr) unsafe.Pointer {
	if alignment == 0 || alignment&(alignment-1) != 0 {
		return nil
	}
	// zeroSizedAlloc need not have the requested alignment.
	if size == 0 {
		size = 1
	}
	padding := uintptr(0)
	if alignment > allocationAlignment {
		padding = alignment - 1
	}
	// Alloc rounds the request to a whole number of GC blocks.
	limit := ^uintptr(0) - (bytesPerBlock - 1)
	if padding > limit || size > limit-padding {
		return nil
	}
	p := Alloc(size + padding)
	return unsafe.Add(p, -uintptr(p)&(alignment-1))
}
