//go:build !go1.26

package maps

import (
	"internal/abi"
	"unsafe"
)

// LLGo's runtime hash functions report unhashable keys once hashing occurs.
// Preserve the existing nil/empty-map behavior here.
func mapKeyError(typ *abi.SwissMapType, p unsafe.Pointer) error {
	return nil
}
