//go:build nogc || baremetal || wasm

package runtime

import "unsafe"

// AllocNoScanU allocates uninitialized memory that must never contain pointers.
// Collectors without a separate pointer-free allocation path use AllocU.
func AllocNoScanU(size uintptr) unsafe.Pointer {
	return AllocU(size)
}
