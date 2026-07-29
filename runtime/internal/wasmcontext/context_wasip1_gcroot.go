//go:build llgo && wasip1 && wasm && !llgo.wasi_threads && llgo_wasm_gc

package wasmcontext

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/internal/gcroot"
)

func (ctx *Context) Resume(nextRoots unsafe.Pointer) {
	gcroot.SwitchAtBoundary((*gcroot.Context)(nextRoots))
	if !ctx.launched {
		contextLaunch(ctx)
		ctx.launched = true
		return
	}
	contextRewind(ctx)
}

func (ctx *Context) Suspend(nextRoots unsafe.Pointer) {
	// Asyncify replays this call stack while rewinding. The owner transition
	// belongs only to the original unwind, not to that replay.
	if contextRewinding() == 0 {
		gcroot.SwitchAtBoundary((*gcroot.Context)(nextRoots))
	}
	contextUnwind(ctx)
}

//go:linkname contextRewinding C.__llgo_wasm_context_rewinding_state
func contextRewinding() uint32

const LLGoFiles = "_asm/context_wasm.S; _asm/context_wasm_gcroot.S"
