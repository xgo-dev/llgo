//go:build linux && !baremetal

package runtime

import (
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

var timerSchedGen uint32

func initTimerSchedulerSync() {
	timerSchedulerMu.Init(nil)
}

func timerSchedulerLock()   { timerSchedulerMu.Lock() }
func timerSchedulerUnlock() { timerSchedulerMu.Unlock() }

// Lost-wakeup invariant for the Linux futex address-wait path:
// timerSchedulerSignalLocked increments timerSchedGen while holding
// timerSchedulerMu, and timerSchedulerWait{Indefinitely,Timed} load gen
// BEFORE releasing timerSchedulerMu. Any signal that mutates the heap and
// calls WakeUint32 therefore either:
//   - happens before unlock (the waiter's WaitUint32 sees timerSchedGen != gen
//     and returns immediately without sleeping), or
//   - acquires timerSchedulerMu after unlock, increments timerSchedGen, and
//     calls WakeUint32 to wake the sleeping waiter.
func timerSchedulerSignalLocked() {
	atomic.Add(&timerSchedGen, uint32(1))
	psync.WakeUint32(&timerSchedGen)
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
	gen := atomic.Load(&timerSchedGen)
	timerSchedulerMu.Unlock()
	timerSchedWaitBackoff(psync.WaitUint32(&timerSchedGen, gen))
	timerSchedulerMu.Lock()
}

func timerSchedulerWaitTimed(wait int64) {
	gen := atomic.Load(&timerSchedGen)
	timerSchedulerMu.Unlock()
	if wait > 0 {
		timerSchedWaitBackoff(psync.WaitUint32Timeout(&timerSchedGen, gen, uint64(wait)))
	}
	timerSchedulerMu.Lock()
}
