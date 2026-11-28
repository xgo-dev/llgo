//go:build !darwin && !linux && !baremetal && (!wasm || (wasip1 && llgo.wasi_threads))

package runtime

func initTimerSchedulerSync() {
	timerSchedulerMu.Init(nil)
	initTimerSchedulerCond()
}

func timerSchedulerLock()   { timerSchedulerMu.Lock() }
func timerSchedulerUnlock() { timerSchedulerMu.Unlock() }

func timerSchedulerSignalLocked() {
	timerSchedulerCond.Signal()
}

func timerSchedulerWaitIndefinitely() {
	timerSchedulerCond.Wait(&timerSchedulerMu)
}

func timerSchedulerWaitTimed(wait int64) {
	timerSchedulerTimedWait(wait)
}
