//go:build llgo && js && wasm && llgo.wasm.workers

package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/wasmworkers"
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

// ThreadDeferSlot returns the current goroutine's defer chain slot.
// Scheduler system-stack work on WebAssembly workers runs with getg() == nil
// and cannot install Go defer frames (panicking via AssertNilDeref would
// recursively dereference nil g inside Panic). Fail fast instead.
func ThreadDeferSlot() **Defer {
	gp := getg()
	if gp == nil {
		fatal("runtime: defer without a current goroutine")
		return nil
	}
	return (**Defer)(unsafe.Add(unsafe.Pointer(gp), unsafe.Offsetof(gp.defer_)))
}

func currentWasmWorker() *wasmWorker {
	return (*wasmWorker)(wasmworkers.Current())
}

func setCurrentWasmWorker(worker *wasmWorker) {
	wasmworkers.SetCurrent(unsafe.Pointer(worker))
}
