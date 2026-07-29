//go:build !wasm || !llgo_wasm_gc

package runtime

import "unsafe"

// Defer presents defer statements in a function.
type Defer struct {
	Addr unsafe.Pointer // sigjmpbuf
	Bits uintptr
	Link *Defer
	Reth unsafe.Pointer // block address after Rethrow
	Rund unsafe.Pointer // block address after RunDefers
	Args unsafe.Pointer // defer func and args links
}

// SetDeferGCRoot is omitted by the compiler when root publication is disabled.
func SetDeferGCRoot(*Defer) {}
