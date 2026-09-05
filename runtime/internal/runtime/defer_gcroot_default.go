//go:build !wasm || !llgo.wasm.gc.linear

package runtime

import "unsafe"

// Defer presents defer statements in a function.
type Defer struct {
	Addr unsafe.Pointer // sigjmpbuf
	Bits uintptr
	Link *Defer
	Reth unsafe.Pointer // native block address or wasm continuation selector
	Rund unsafe.Pointer // native block address or wasm continuation selector
	Args unsafe.Pointer // defer func and args links
}
