//go:build wasm && llgo_wasm_gc

package runtime

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/internal/gcroot"
)

// Defer presents defer statements in a function.
type Defer struct {
	Addr   unsafe.Pointer // sigjmpbuf
	Bits   uintptr
	Link   *Defer
	Reth   unsafe.Pointer // block address after Rethrow
	Rund   unsafe.Pointer // block address after RunDefers
	Args   unsafe.Pointer // defer func and args links
	gcRoot unsafe.Pointer // root chain at the owning function's setjmp
}

// SetDeferGCRoot records the chain that longjmp must restore.
func SetDeferGCRoot(frame *Defer) {
	frame.gcRoot = gcroot.CurrentChain()
}
