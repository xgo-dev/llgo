//go:build llgo && wasm && wasip1 && llgo.wasi_threads

package runtime

import c "github.com/xgo-dev/llgo/runtime/internal/clite"

// Keep the initial Wasm invocation alive until the final worker reports the
// Goexit deadlock. Returning from the entry point can finish the host process
// before its workers; this parked thread no longer participates in Go GC.
func parkInitialWasiThread(gp *g) {
	releaseStartArg(gp)
	freeChanWaiter(gp)
	freeSelectCache(gp)
	casgstatus(gp, _Grunning, _Gdead)
	releaseGAndCheckDeadlock()
	// This thread will never return to Go. Leaving it registered would make
	// every later worker collection time out waiting for its safepoint.
	unregisterWasiGCThread()
	for {
		c.Usleep(1000)
	}
}
