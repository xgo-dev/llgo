//go:build nogc || baremetal || wasm

package runtime

import "unsafe"

// AllocAtomicU allocates uninitialized memory that must never contain pointers.
// Collectors without a separate pointer-free allocation path use AllocU.
func AllocAtomicU(size uintptr) unsafe.Pointer {
	return AllocU(size)
}

// AllocAtomicZ allocates zero-initialized memory that must never contain pointers.
// Use the backend's zeroing allocator so collectors that already clear storage
// do not clear it a second time.
func AllocAtomicZ(size uintptr) unsafe.Pointer {
	return AllocZ(size)
}
