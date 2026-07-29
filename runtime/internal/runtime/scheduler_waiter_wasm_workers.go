//go:build llgo && js && wasm && llgo.wasm_workers

package runtime

import "github.com/goplus/llgo/runtime/internal/clite/sync/atomic"

func tryCasgstatus(gp *g, oldval, newval uint32) bool {
	_, ok := atomic.CompareAndExchange(&gp.atomicstatus, oldval, newval)
	return ok
}

// SchedulerWaiter is an opaque one-shot notification owned by a waiting G.
// The notification closes the Ready-before-Park race between Web workers.
type SchedulerWaiter struct {
	gp       *g
	notified uint32
}

func CurrentSchedulerWaiter() SchedulerWaiter {
	return SchedulerWaiter{gp: getg()}
}

func (w *SchedulerWaiter) Park() {
	gp := w.gp
	if gp == nil || getg() != gp {
		fatal("runtime: invalid WebAssembly scheduler waiter")
		return
	}
	if _, ok := atomic.CompareAndExchange(&w.notified, uint32(1), uint32(0)); ok {
		return
	}

	casgstatus(gp, _Grunning, _Gwaiting)
	// Keep the G active until an early notification is either consumed here
	// or has made the G runnable. Otherwise worker 0 can observe a transient
	// zero active count and report a false deadlock.
	if _, ok := atomic.CompareAndExchange(&w.notified, uint32(1), uint32(0)); ok {
		if tryCasgstatus(gp, _Gwaiting, _Grunning) {
			return
		}
	}

	atomic.Add(&wasmMultiSched.active, ^uint32(0))
	wakeWasmEventWorker()
	worker := gp.context.platform.owner
	gp.context.platform.context.Swap(
		&worker.system,
		wasmWorkerSystemRootPointer(worker),
	)
	atomic.Store(&w.notified, uint32(0))
}

func (w *SchedulerWaiter) Ready() {
	gp := w.gp
	if gp == nil {
		fatal("runtime: ready of invalid WebAssembly scheduler waiter")
		return
	}
	if _, ok := atomic.CompareAndExchange(&w.notified, uint32(0), uint32(1)); !ok {
		return
	}
	if tryCasgstatus(gp, _Gwaiting, _Grunnable) {
		atomic.Add(&wasmMultiSched.active, uint32(1))
		enqueueWasmG(gp.context.platform.owner, gp)
	}
}

func SchedulerProcID() int {
	worker := currentWasmWorker()
	if worker == nil {
		return 0
	}
	return worker.index
}
