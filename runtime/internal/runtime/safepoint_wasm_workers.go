//go:build llgo && js && wasm && llgo.wasm_workers

package runtime

import "github.com/goplus/llgo/runtime/internal/wasmevent"

const wasmSafepointQuantum = uint32(1024)

func CooperativeSafepoint() {
	worker := currentWasmWorker()
	if worker == nil || !worker.safepointBudget.Poll() {
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
	if worker.index == 0 {
		wasmevent.Poll()
	}
	if wasmWorkerRunqLen(worker) != 0 {
		goschedBackend()
	}
}
