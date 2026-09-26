//go:build llgo && wasm && wasip1 && llgo.wasi_threads

package runtime

import c "github.com/xgo-dev/llgo/runtime/internal/clite"

// WAMR's pthread_exit does not terminate its initial execution environment.
// Keep that environment alive until the final worker reports the Goexit
// deadlock. A return from pthread_exit would resume the dead main goroutine.
func parkInitialWasiThread(gp *g) {
	releaseStartArg(gp)
	casgstatus(gp, _Grunning, _Gdead)
	releaseGAndCheckDeadlock()
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
