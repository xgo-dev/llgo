//go:build darwin && !baremetal

package runtime

import (
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
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

// Lost-wakeup invariant for the Darwin address-wait path:
// timerSchedulerSignalLocked increments timerSchedGen while holding
// timerSchedUnfairMu, and timerSchedulerWait{Indefinitely,Timed} load gen
// BEFORE releasing timerSchedUnfairMu. Any signal that mutates the heap and
// calls WakeUint32 therefore either:
//   - happens before unlock (the waiter's WaitUint32 sees timerSchedGen != gen
//     and returns immediately without sleeping), or
//   - acquires timerSchedUnfairMu after unlock, increments timerSchedGen, and
//     calls WakeUint32 to wake the sleeping waiter.
func timerSchedulerSignalLocked() {
	if timerSchedUseFast {
		atomic.Add(&timerSchedGen, uint32(1))
		psync.WakeUint32(&timerSchedGen)
		return
	}
	timerSchedulerCond.Signal()
}

func timerSchedWaitBackoff(err c.Int) {
	if psync.WaitErrnoRetryable(err) {
		return
	}
	// Prevent a tight 100%-CPU spin if the kernel address wait fails
	// persistently (e.g. ENOMEM/EFAULT).
	c.Usleep(100)
}

func timerSchedulerWaitIndefinitely() {
	if timerSchedUseFast {
		gen := atomic.Load(&timerSchedGen)
		timerSchedUnfairMu.Unlock()
		timerSchedWaitBackoff(psync.WaitUint32(&timerSchedGen, gen))
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
			timerSchedWaitBackoff(psync.WaitUint32Timeout(&timerSchedGen, gen, uint64(wait)))
		}
		timerSchedUnfairMu.Lock()
		return
	}
	timerSchedulerTimedWait(wait)
}
