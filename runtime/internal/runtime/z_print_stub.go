//go:build !llgo && !wasm

package runtime

import "unsafe"

// PrintString32 keeps compiler-only host tests type-correct.
func PrintString32(unsafe.Pointer, uint32) {}
