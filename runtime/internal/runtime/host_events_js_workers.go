//go:build llgo && js && wasm && llgo.wasm.workers

package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/wasmworkers"
)

// A synchronous JavaScript callback resumes the same goroutine that entered
// its JavaScript call. Keep the event stack per physical worker: the callback
// can temporarily park while other goroutines satisfy a channel operation.
type wasmJSEvent struct {
	gp       *g
	id       uint32
	returned bool
	resume   bool
}

type wasmExternalJSEvent struct {
	gp   *g
	done bool
}

func HandleWasmEvent(handler func()) {
	worker := currentWasmWorker()
	if worker == nil {
		fatal("runtime: JavaScript callback without a worker")
		return
	}
	if getg() == nil {
		// External JS events arrive after the worker has returned to its
		// event loop. Adopt this export's native stack as the system fiber and
		// run its callback G until it completes or first suspends.
		worker.system.ResetCurrent()
		resumeWasmWorkerGCSystem(worker)
		worker.syncEventDepth++
		e := &wasmExternalJSEvent{}
		go func() {
			e.gp = getg()
			handler()
			e.done = true
		}()
		for !e.done {
			gp := waitWasmWorkerRunq(worker)
			if gp == nil {
				fatal("runtime: synchronous JavaScript callback stalled")
				break
			}
			casgstatus(gp, _Grunnable, _Grunning)
			runWasmG(worker, gp)
			wasmWorkerStopForGC(worker)
			if readgstatus(gp) == _Gdead {
				releaseWasmContext(gp)
			}
			if gp == e.gp && !e.done {
				// Like Go's JS event bridge, a callback that suspends returns
				// undefined to JS. Resume the remaining Go work after the host
				// event stack has unwound, so another JS call can interleave.
				worker.syncEventDepth--
				suspendWasmWorkerGCSystem(worker)
				wasmworkers.ResumeSoon(unsafe.Pointer(worker))
				return
			}
		}
		worker.syncEventDepth--
		suspendWasmWorkerGCSystem(worker)
		return
	}
	worker.nextJSEventID++
	if worker.nextJSEventID == 0 {
		worker.nextJSEventID++
	}
	gp := getg()
	// Give a newly entered handler one resumption before unrelated JS callers
	// can open more native callback stacks. Later yields alternate with Go work;
	// nested events keep their own resume intent.
	e := &wasmJSEvent{gp: gp, id: worker.nextJSEventID, resume: true}
	previousCallbackID := gp.context.platform.jsCallbackID
	gp.context.platform.jsCallbackID = e.id
	worker.jsEvents = append(worker.jsEvents, e)
	handler()
	e.returned = true
	gopark()
	worker.jsEvents[len(worker.jsEvents)-1] = nil
	worker.jsEvents = worker.jsEvents[:len(worker.jsEvents)-1]
	gp.context.platform.jsCallbackID = previousCallbackID
}

func PollWasmEvent() {
	worker := currentWasmWorker()
	if worker == nil || getg() != nil || len(worker.jsEvents) == 0 {
		return
	}
	e := worker.jsEvents[len(worker.jsEvents)-1]
	if e.returned && readgstatus(e.gp) == _Gwaiting {
		goready(e.gp)
	}
}
