//go:build llgo && !baremetal && !wasm

package runtime

import (
	"github.com/xgo-dev/llgo/runtime/internal/stacktrace"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

// TracebackCreator is installed by the public runtime's symbol-aware walker.
var TracebackCreator func() uintptr

func registerTraceback(gp, parent *g) {
	if parent != nil && TracebackCreator != nil {
		gp.createdPC = TracebackCreator()
	}
	gp.tracebackThread = stacktrace.Register(gp.goid, gp.parentGoid, gp.createdPC)
}

func attachTraceback(gp *g) {
	if gp != nil {
		stacktrace.Attach(gp.tracebackThread)
	}
}

func unregisterTraceback(gp *g) {
	if gp.tracebackThread != nil {
		stacktrace.Unregister(gp.tracebackThread)
		gp.tracebackThread = nil
	}
}

func releaseFaultSnapshot() { stacktrace.ReleaseFaultBuffer() }

func TracebackWaiting(waiting bool) {
	gp := getg()
	state := uint32(_Grunning)
	if waiting {
		state = _Gwaiting
	}
	atomic.Store(&gp.atomicstatus, state)
	stacktrace.SetState(gp.tracebackThread, state)
}
