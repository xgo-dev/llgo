//go:build wasm

package runtime

import "unsafe"

// FunctionPC leaves the low two bits available for caller shadow-stack PCs.
// Raw wasm function pointers are indirect-function table indices and may have
// any alignment, unlike native text addresses.
func FunctionPC(ptr unsafe.Pointer) uintptr {
	return uintptr(ptr) << 2
}
