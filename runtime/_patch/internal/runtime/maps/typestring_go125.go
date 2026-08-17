//go:build go1.25 && !go1.26

package maps

import (
	"internal/abi"
	_ "unsafe"
)

func typeString(typ *abi.Type) string {
	if typ == nil {
		return "<nil>"
	}
	return llgoTypeString(typ)
}

//go:linkname llgoTypeString github.com/xgo-dev/llgo/runtime/abi.(*Type).String
func llgoTypeString(typ *abi.Type) string
