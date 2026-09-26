//go:build llgo && js && wasm && llgo.wasm.workers && llgo.wasm.gc.linear

package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
	"github.com/xgo-dev/llgo/runtime/internal/wasmworkers"
)

const noWasmGCOwner = ^uint32(0)

type wasmWorkerGCState struct {
	systemRoot wasmGCRootContext
	epoch      uint32
	ready      uint32
}

var wasmGCWorld struct {
	epoch   uint32
	stopped uint32
	ready   uint32
	owner   uint32
}

func initWasmWorkerGCSystem(worker *wasmWorker) {
	if worker.index == 0 {
		registerWasmGCRoot(&worker.gc.systemRoot, false)
		// InitCurrent adopts this call stack as the system fiber without a
		// stack switch, so its compiler root chain remains installed.
		adoptWasmGCRoot(&worker.gc.systemRoot)
	} else {
		registerWasmGCRoot(&worker.gc.systemRoot, true)
	}
	atomic.Store(&worker.gc.ready, uint32(1))
	atomic.Add(&wasmGCWorld.ready, uint32(1))
	wasmWorkerStopForGC(worker)
}

func suspendWasmWorkerGCSystem(worker *wasmWorker) {
	suspendWasmGCRoot(&worker.gc.systemRoot)
}

func resumeWasmWorkerGCSystem(worker *wasmWorker) {
	adoptWasmGCRoot(&worker.gc.systemRoot)
}

func wasmWorkerSystemRootPointer(worker *wasmWorker) unsafe.Pointer {
	return wasmGCRootPointer(&worker.gc.systemRoot)
}

func wasmGCRequestPending(worker *wasmWorker) bool {
	epoch := atomic.Load(&wasmGCWorld.epoch)
	return epoch&1 != 0 && wasmGCWorldOwner(worker) != atomic.Load(&wasmGCWorld.owner)
}

func wasmGCOwnedBy(worker *wasmWorker) bool {
	epoch := atomic.Load(&wasmGCWorld.epoch)
	return epoch&1 != 0 && wasmGCWorldOwner(worker) == atomic.Load(&wasmGCWorld.owner)
}

func wasmWorkerStopForGC(worker *wasmWorker) bool {
	stopped := false
	for {
		epoch := atomic.Load(&wasmGCWorld.epoch)
		if epoch&1 == 0 || wasmGCWorldOwner(worker) == atomic.Load(&wasmGCWorld.owner) {
			return stopped
		}
		if worker.gc.epoch != epoch {
			worker.gc.epoch = epoch
			publishWasmGCRoot()
			atomic.Add(&wasmGCWorld.stopped, uint32(1))
			wasmworkers.Wake(&wasmGCWorld.stopped)
		}
		for atomic.Load(&wasmGCWorld.epoch) == epoch {
			wasmworkers.Wait(&wasmGCWorld.epoch, epoch, -1)
		}
		stopped = true
	}
}

func wasmGCStopTheWorld() {
	worker := currentWasmWorker()
	owner := wasmGCWorldOwner(worker)
	atomic.Store(&wasmGCWorld.owner, owner)
	atomic.Store(&wasmGCWorld.stopped, uint32(0))
	// LLGo's atomic.Add returns the value before the addition.
	epoch := atomic.Add(&wasmGCWorld.epoch, uint32(1)) + 1
	if epoch&1 == 0 {
		fatal("runtime: overlapping WebAssembly GC cycles")
		return
	}
	wakeAllWasmWorkers()

	target := atomic.Load(&wasmGCWorld.ready)
	if worker != nil && atomic.Load(&worker.gc.ready) != 0 {
		target--
	}
	for {
		stopped := atomic.Load(&wasmGCWorld.stopped)
		if stopped >= target {
			break
		}
		wasmworkers.Wait(&wasmGCWorld.stopped, stopped, -1)
	}
}

func wasmGCResumeWorld() {
	epoch := atomic.Load(&wasmGCWorld.epoch)
	if epoch&1 == 0 {
		fatal("runtime: resumed a running WebAssembly world")
		return
	}
	atomic.Store(&wasmGCWorld.epoch, epoch+1)
	atomic.Store(&wasmGCWorld.owner, noWasmGCOwner)
	wasmworkers.Wake(&wasmGCWorld.epoch)
	wakeAllWasmWorkers()
}

func wasmGCAllocatorYield() {
	worker := currentWasmWorker()
	if worker == nil {
		return
	}
	// This callback runs while an allocation is waiting for the GC mutex and
	// can therefore sit below a foreign C/C++ frame. It may acknowledge an STW
	// request in place, but must not schedule another G: doing so asks Asyncify
	// to unwind and replay the active foreign allocation call, whose native
	// locals are not a Go fiber continuation. The stopped root is published by
	// wasmWorkerStopForGC and the same G resumes after the collector releases
	// the world.
	wasmWorkerStopForGC(worker)
}

func wasmGCWorldOwner(worker *wasmWorker) uint32 {
	if worker == nil {
		return noWasmGCOwner
	}
	return uint32(worker.index)
}

func wakeAllWasmWorkers() {
	for i := 0; i < wasmMultiSched.count; i++ {
		wakeWasmWorker(&wasmMultiSched.workers[i])
	}
}
