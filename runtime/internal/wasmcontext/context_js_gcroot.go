//go:build llgo && js && wasm && llgo.wasm.gc.linear

package wasmcontext

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/clite/emscripten"
	"github.com/xgo-dev/llgo/runtime/internal/gcroot"
)

func (ctx *Context) Swap(next *Context, nextRoots unsafe.Pointer) {
	// This wrapper is replayed while Emscripten rewinds the destination
	// fiber. Switch ownership only on the original unwind; replayed function
	// entries reconstruct the destination's compiler root chain from nil.
	if gcroot.Rebuilding() && !Rewinding() {
		// A previous replay can return through host-generated Asyncify code
		// without executing the Go continuation after FiberSwap.
		gcroot.FinishRebuild()
	}
	if !gcroot.Rebuilding() {
		gcroot.BeginRebuild((*gcroot.Context)(nextRoots))
	}
	emscripten.FiberSwap(&ctx.fiber, &next.fiber)
	gcroot.FinishRebuild()
}

// Rewinding reports the actual Emscripten Asyncify state.
func Rewinding() bool {
	return emscripten.FiberRewinding() != 0
}
