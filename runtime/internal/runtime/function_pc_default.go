//go:build !wasm

package runtime

import "unsafe"

// FunctionPC converts a callable code pointer to the PC identity exposed by
// reflect and runtime symbolization.
func FunctionPC(ptr unsafe.Pointer) uintptr {
	return uintptr(ptr)
}
