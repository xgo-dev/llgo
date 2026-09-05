//go:build wasm && llgo.wasm.gc.linear

package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/gcroot"
)

// Defer presents defer statements in a function.
type Defer struct {
	Addr   unsafe.Pointer // sigjmpbuf
	Bits   uintptr
	Link   *Defer
	Reth   unsafe.Pointer // native block address or wasm continuation selector
	Rund   unsafe.Pointer // native block address or wasm continuation selector
	Args   unsafe.Pointer // defer func and args links
	gcRoot unsafe.Pointer // compiler root chain at this function's setjmp
	gcNext unsafe.Pointer // saved next link of the setjmp function's root frame
	gcMap  unsafe.Pointer // saved map of the setjmp function's root frame
}

// SetDeferGCRoot records the caller chain that longjmp must restore. The
// compiler passes the chain explicitly because this helper's own root frame
// may already be active while the call executes.
func SetDeferGCRoot(frame *Defer, chain unsafe.Pointer) {
	frame.gcRoot = chain
	if frame.gcRoot != nil {
		frame.gcNext = *gcRootHeaderField(frame.gcRoot, 0)
		frame.gcMap = *gcRootHeaderField(frame.gcRoot, 1)
	}
}

// RestoreDeferGCRoot repairs the setjmp function's root-frame header after a
// wasm longjmp replay. LLVM's SJLJ transform can reuse its stack slots while
// returning to setjmp, so the compiler calls this immediately after both the
// initial and non-local returns.
func RestoreDeferGCRoot(frame *Defer) {
	if frame.gcRoot != nil {
		*gcRootHeaderField(frame.gcRoot, 0) = frame.gcNext
		*gcRootHeaderField(frame.gcRoot, 1) = frame.gcMap
	}
	gcroot.RestoreChain(frame.gcRoot)
	gcroot.FinishSJLJReplay()
}

// gcRootHeaderField addresses the compiler-owned {next, map} frame header.
// Keep this independent of Go struct layout: Emscripten Memory64 uses 64-bit
// LLVM pointers while GOARCH remains wasm, and the compiler frame follows the
// target pointer width selected by the active WASM ABI.
func gcRootHeaderField(root unsafe.Pointer, field uintptr) *unsafe.Pointer {
	return (*unsafe.Pointer)(unsafe.Add(root, field*unsafe.Sizeof(root)))
}
