//go:build go1.26

package maps

import (
	"internal/abi"
	"unsafe"
)

func typeString(typ *abi.Type) string {
	if typ == nil {
		return "<nil>"
	}
	return llgoTypeString(typ)
}

//go:linkname llgoTypeString github.com/xgo-dev/llgo/runtime/abi.(*Type).String
func llgoTypeString(typ *abi.Type) string

// LLGo's runtime hash functions report unhashable keys once hashing occurs.
// Preserve the existing nil/empty-map behavior here.
func mapKeyError(typ *abi.MapType, p unsafe.Pointer) error {
	return nil
}
