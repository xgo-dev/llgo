//go:build llgo && !baremetal && !wasm

package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/stacktrace"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

// TracebackCreator captures raw parent PCs without initializing symbol tables.
var TracebackCreator func([]uintptr) int

func registerTraceback(gp, parent *g) {
	var pcs [stacktrace.MaxCreatorFrames]uintptr
	var n int
	if parent != nil && TracebackCreator != nil {
		n = TracebackCreator(pcs[:])
	}
	gp.tracebackThread = stacktrace.Register(gp.goid, gp.parentGoid, gp.createdPC, &pcs[0], uintptr(n))
}

func TracebackParentPCs() (uint64, []uintptr) {
	gp := getg()
	var n uintptr
	pcs := stacktrace.ThreadCreator(gp.tracebackThread, &n)
	return gp.parentGoid, unsafe.Slice(pcs, n)
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
