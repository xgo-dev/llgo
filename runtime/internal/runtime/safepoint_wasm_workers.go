//go:build llgo && js && wasm && llgo.wasm.workers

package runtime

import (
	"github.com/xgo-dev/llgo/runtime/internal/gcroot"
	"github.com/xgo-dev/llgo/runtime/internal/wasmcontext"
)

const wasmSafepointQuantum = uint32(1024)

func CooperativeSafepoint() {
	// Asyncify has not restored the suspended frame arguments until replay
	// reaches the original fiber swap. Scheduling or acknowledging a GC request
	// from that partial stack corrupts the replay data and can expose stale
	// roots to the collector.
	if gcroot.Rebuilding() {
		if wasmcontext.Rewinding() {
			return
		}
		gcroot.FinishRebuild()
	}
	worker := currentWasmWorker()
	// The collector owns its worker until sweeping has completed. Compiler-
	// inserted polls in GC loops must not reschedule while the allocator and
	// heap metadata are exclusively owned by that worker.
	if worker == nil || wasmGCOwnedBy(worker) ||
		(!wasmGCRequestPending(worker) && !worker.safepointBudget.Poll()) {
		return
	}
	cooperativeSafepointSlow()
}

//go:noinline
func cooperativeSafepointSlow() {
	worker := currentWasmWorker()
	if worker == nil {
		return
	}
	gp := getg()
	if wasmGCRequestPending(worker) {
		if gp != nil {
			gp.context.platform.context.Swap(
				&worker.system,
				wasmWorkerSystemRootPointer(worker),
			)
		} else {
			wasmWorkerStopForGC(worker)
		}
		return
	}
	// Host callback entries and scheduler/GC bookkeeping execute on the system
	// fiber with no current G. They may take the same shared locks as Go code,
	// but cannot enter the goroutine scheduler from inside those lock paths.
	if gp == nil {
		return
	}
	hooks := loadWasmEventHooks()
	hooks.pollCallbackEvents(worker)
	if worker.index == 0 {
		hooks.pollTimerEvents()
	}
	if wasmWorkerRunqLen(worker) != 0 {
		goschedBackend()
	}
}
