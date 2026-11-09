//go:build nogc || baremetal || wasm

package runtime

import "unsafe"

// AllocAtomicU allocates uninitialized memory that must never contain pointers.
// Collectors without a separate pointer-free allocation path use AllocU.
func AllocAtomicU(size uintptr) unsafe.Pointer {
	return AllocU(size)
}
