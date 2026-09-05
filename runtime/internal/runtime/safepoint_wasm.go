//go:build llgo && wasm && llgo.wasm.gc.linear && !(wasip1 && llgo.wasi_threads)

package runtime

import (
	"github.com/xgo-dev/llgo/runtime/internal/gcroot"
	"github.com/xgo-dev/llgo/runtime/internal/pollbudget"
	"github.com/xgo-dev/llgo/runtime/internal/wasmcontext"
)

const wasmSafepointQuantum = uint32(1024)

var wasmSafepointBudget = pollbudget.New(wasmSafepointQuantum)

// CooperativeSafepoint gives the single wasm worker a bounded opportunity to
// run host events and another runnable goroutine.
func CooperativeSafepoint() {
	// Asyncify's replay stack is incomplete until the suspension call is
	// reached, so neither scheduling nor collection is safe during that phase.
	if gcroot.Rebuilding() {
		if wasmcontext.Rewinding() {
			return
		}
		gcroot.FinishRebuild()
	}
	if !wasmSafepointBudget.Poll() {
		return
	}
	cooperativeSafepointSlow()
}

//go:noinline
func cooperativeSafepointSlow() {
	if !wasmSched.started {
		return
	}
	pollWasmEvents()
	if wasmSched.runq.Len() != 0 {
		goschedBackend()
	}
}
