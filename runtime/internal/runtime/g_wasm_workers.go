//go:build llgo && js && wasm && llgo.wasm_workers

package runtime

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/internal/wasmworkers"
)

func getg() *g {
	if worker := currentWasmWorker(); worker != nil {
		return worker.m.curg
	}
	if wasmMultiSched.started {
		return nil
	}
	return initRuntimeContext(allocRuntimeContext(), nil, _Grunning)
}

func setg(gp *g) {
	worker := currentWasmWorker()
	if worker == nil {
		fatal("runtime: setg without a WebAssembly worker")
		return
	}
	worker.m.curg = gp
}

func currentWasmWorker() *wasmWorker {
	return (*wasmWorker)(wasmworkers.Current())
}

func setCurrentWasmWorker(worker *wasmWorker) {
	wasmworkers.SetCurrent(unsafe.Pointer(worker))
}
