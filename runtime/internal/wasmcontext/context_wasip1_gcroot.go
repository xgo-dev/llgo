//go:build llgo && wasip1 && wasm && !llgo.wasi_threads && llgo.wasm.gc.linear

package wasmcontext

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/gcroot"
)

func (ctx *Context) Resume(nextRoots unsafe.Pointer) {
	// Asyncify replays function entries to restore this context. Start with an
	// empty chain so those entries reconstruct it instead of linking restored
	// frames to themselves.
	if gcroot.Rebuilding() && !Rewinding() {
		gcroot.FinishRebuild()
	}
	gcroot.BeginRebuild((*gcroot.Context)(nextRoots))
	if !ctx.launched {
		contextLaunch(ctx)
		ctx.launched = true
		return
	}
	contextRewind(ctx)
	gcroot.FinishRebuild()
}

func (ctx *Context) Suspend(nextRoots unsafe.Pointer) {
	// Asyncify replays this call stack while rewinding. The owner transition
	// belongs only to the original unwind, not to that replay.
	if Rewinding() {
		// Reaching the suspension boundary means Asyncify has reconstructed
		// every caller frame. Complete the rebuild before stopping rewind;
		// code after contextUnwind is skipped by the Asyncify transform.
		gcroot.FinishRebuild()
	} else {
		gcroot.SwitchAtBoundary((*gcroot.Context)(nextRoots))
	}
	contextUnwind(ctx)
}

// Rewinding reports the actual Binaryen Asyncify state.
func Rewinding() bool {
	return contextRewinding() != 0
}

//go:linkname contextRewinding C.__llgo_wasm_context_rewinding_state
func contextRewinding() uint32

const LLGoFiles = "_asm/context_wasm.S; _asm/context_wasm_gcroot.S"
