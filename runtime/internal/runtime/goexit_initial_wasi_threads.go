//go:build llgo && wasm && wasip1 && llgo.wasi_threads

package runtime

import c "github.com/xgo-dev/llgo/runtime/internal/clite"

// Keep the initial Wasm invocation alive until the final worker reports the
// Goexit deadlock. Returning from the entry point can finish the host process
// before its workers; this parked thread no longer participates in Go GC.
func parkInitialWasiThread(gp *g) {
	releaseStartArg(gp)
	casgstatus(gp, _Grunning, _Gdead)
	releaseGAndCheckDeadlock()
	// This thread will never return to Go. Leaving it registered would make
	// every later worker collection time out waiting for its safepoint.
	unregisterWasiGCThread()
	for {
		c.Usleep(1000)
	}
}

// MarkTimerSystemGoroutine excludes the persistent timer pthread from the
// user-goroutine count. Otherwise main Goexit can wait forever after the last
// user G returns because the idle timer service remains registered.
func MarkTimerSystemGoroutine() {
	gp := getg()
	if gp.context.platform.systemG {
		return
	}
	gp.context.platform.systemG = true
	releaseGAndCheckDeadlock()
}
