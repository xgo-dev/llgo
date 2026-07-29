//go:build llgo && js && wasm && llgo.wasm_workers

package runtime

import "github.com/goplus/llgo/runtime/internal/wasmevent"

const wasmSafepointQuantum = uint32(1024)

func CooperativeSafepoint() {
	worker := currentWasmWorker()
	if worker == nil || (!wasmGCRequestPending(worker) && !worker.safepointBudget.Poll()) {
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
	if wasmGCRequestPending(worker) {
		if gp := getg(); gp != nil {
			gp.context.platform.context.Swap(
				&worker.system,
				wasmWorkerSystemRootPointer(worker),
			)
		} else {
			wasmWorkerStopForGC(worker)
		}
		return
	}
	if worker.index == 0 {
		wasmevent.Poll()
	}
	if wasmWorkerRunqLen(worker) != 0 {
		goschedBackend()
	}
}
