//go:build llgo && js && wasm && llgo.wasm.workers

package runtime

import (
	"unsafe"

	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
	"github.com/xgo-dev/llgo/runtime/internal/wasmsync"
	"github.com/xgo-dev/llgo/runtime/internal/wordmap"
)

//go:linkname registerWasmTimerHooks github.com/xgo-dev/llgo/runtime/internal/runtime.RegisterWasmTimerHooks
func registerWasmTimerHooks(poll func(), wait func() (uint64, bool))

var (
	timerSchedulerHeap []*timerState
	timerSchedulerMap  wordmap.Map[unsafe.Pointer, *timerState]
	timerSchedulerLock wasmsync.Mutex
)

func init() {
	registerWasmTimerHooks(wasmPollTimers, wasmTimerWait)
}

func lockTimerScheduler() {
	timerSchedulerLock.Lock(llruntime.CooperativeSafepoint)
}

func startRuntimeTimer(r *runtimeTimer) {
	if r != nil && startSynctestTimer(r) {
		return
	}
	if r == nil {
		return
	}
	lockTimerScheduler()
	st := timerSchedulerMap.Get(unsafe.Pointer(r))
	if st == nil {
		st = &timerState{r: r, heapIndex: -1}
		timerSchedulerMap.Set(unsafe.Pointer(r), st)
	} else if st.active {
		timerHeapRemove(st.heapIndex)
	}
	st.callback = snapshotRuntimeTimer(r)
	st.active = true
	timerHeapAdd(st)
	timerSchedulerLock.Unlock()
	llruntime.WakeWasmScheduler()
}

func stopRuntimeTimer(r *runtimeTimer) bool {
	if r != nil && r.fake != nil {
		return r.fake.Stop()
	}
	if r == nil {
		return false
	}
	lockTimerScheduler()
	st := timerSchedulerMap.Get(unsafe.Pointer(r))
	wasActive := st != nil && st.active
	if wasActive {
		timerHeapRemove(st.heapIndex)
		st.active = false
		timerSchedulerMap.Delete(unsafe.Pointer(r))
	}
	timerSchedulerLock.Unlock()
	if wasActive {
		llruntime.WakeWasmScheduler()
	}
	return wasActive
}

func resetRuntimeTimer(r *runtimeTimer, when, period int64, update func()) bool {
	if r != nil && r.fake != nil {
		r.fake.Check()
		if update != nil {
			update()
		}
		r.when, r.period = when, period
		callback := snapshotRuntimeTimer(r)
		return r.fake.Reset(when, period, callback.run)
	}
	if r == nil {
		return false
	}
	lockTimerScheduler()
	st := timerSchedulerMap.Get(unsafe.Pointer(r))
	wasActive := st != nil && st.active
	if st == nil {
		st = &timerState{r: r, heapIndex: -1}
		timerSchedulerMap.Set(unsafe.Pointer(r), st)
	} else if st.active {
		timerHeapRemove(st.heapIndex)
	}
	if update != nil {
		update()
	}
	r.when = when
	r.period = period
	st.callback = snapshotRuntimeTimer(r)
	st.active = true
	timerHeapAdd(st)
	timerSchedulerLock.Unlock()
	llruntime.WakeWasmScheduler()
	return wasActive
}

func wasmTimerWait() (wait uint64, active bool) {
	lockTimerScheduler()
	if len(timerSchedulerHeap) == 0 {
		timerSchedulerLock.Unlock()
		return 0, false
	}
	when := timerSchedulerHeap[0].r.when
	timerSchedulerLock.Unlock()
	now := runtimeNano()
	if when <= now {
		return 0, true
	}
	return uint64(when - now), true
}

func wasmPollTimers() {
	for {
		now := runtimeNano()
		lockTimerScheduler()
		if len(timerSchedulerHeap) == 0 || timerSchedulerHeap[0].r.when > now {
			timerSchedulerLock.Unlock()
			return
		}
		st := timerSchedulerHeap[0]
		when := st.r.when
		period := st.r.period
		callback := st.callback
		timerHeapRemove(0)
		if period > 0 {
			st.r.when = timerNextWhen(when, period, now)
			timerHeapAdd(st)
		} else {
			st.active = false
			timerSchedulerMap.Delete(unsafe.Pointer(st.r))
		}
		timerSchedulerLock.Unlock()

		delay := now - when
		if delay < 0 {
			delay = 0
		}
		callback.run(delay)
	}
}
