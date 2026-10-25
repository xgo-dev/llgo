//go:build windows && 386

package reflect

import (
	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llgo/runtime/internal/ffi"
)

// Windows/386 C ABI removes Go field wrappers and their tail padding:
// two pointers followed by struct{} occupy 12 bytes in Go storage but
// only 8 bytes at the call boundary. Keep the original field description.
func padFFIZeroSizedStructTail(_ *abi.Type, fields []*ffi.Type, _ uintptr) []*ffi.Type {
	return fields
}
