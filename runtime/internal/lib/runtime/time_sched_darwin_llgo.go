//go:build darwin && !baremetal

package runtime

import (
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

var (
	timerSchedUseFast  bool
	timerSchedUnfairMu psync.UnfairMutex
	timerSchedGen      uint32
)

func initTimerSchedulerSync() {
	timerSchedUseFast = psync.HasWaitUint32Timeout()
	if !timerSchedUseFast {
		timerSchedulerMu.Init(nil)
		initTimerSchedulerCond()
	}
}

func timerSchedulerLock() {
	if timerSchedUseFast {
		timerSchedUnfairMu.Lock()
		return
	}
	timerSchedulerMu.Lock()
}

func timerSchedulerUnlock() {
	if timerSchedUseFast {
		timerSchedUnfairMu.Unlock()
		return
	}
	timerSchedulerMu.Unlock()
}

func timerSchedulerSignalLocked() {
	if timerSchedUseFast {
		atomic.Add(&timerSchedGen, uint32(1))
		psync.WakeUint32(&timerSchedGen)
		return
	}
	timerSchedulerCond.Signal()
}

func timerSchedulerWaitIndefinitely() {
	if timerSchedUseFast {
		gen := atomic.Load(&timerSchedGen)
		timerSchedUnfairMu.Unlock()
		psync.WaitUint32(&timerSchedGen, gen)
		timerSchedUnfairMu.Lock()
		return
	}
	timerSchedulerCond.Wait(&timerSchedulerMu)
}

func timerSchedulerWaitTimed(wait int64) {
	if timerSchedUseFast {
		gen := atomic.Load(&timerSchedGen)
		timerSchedUnfairMu.Unlock()
		if wait > 0 {
			psync.WaitUint32Timeout(&timerSchedGen, gen, uint64(wait))
		}
		timerSchedUnfairMu.Lock()
		return
	}
	timerSchedulerTimedWait(wait)
}
