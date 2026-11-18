//go:build baremetal

package runtime

import "unsafe"

// The baremetal task backend runs in a single execution context. Plain pointer
// loads and stores avoid unavailable libatomic helpers on small RISC-V targets.
func loadItabPointer(ptr *unsafe.Pointer) unsafe.Pointer {
	return *ptr
}

func storeItabPointer(ptr *unsafe.Pointer, value unsafe.Pointer) {
	*ptr = value
}
