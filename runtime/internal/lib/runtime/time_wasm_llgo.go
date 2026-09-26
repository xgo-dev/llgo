//go:build wasm && !baremetal && !llgo.wasm.workers && !(wasip1 && llgo.wasi_threads)

package runtime

import _ "unsafe"

// The single-worker WebAssembly runtime is non-reentrant: host waits unwind
// to Node/the WASI host and resume before another timer mutation can run. It
// therefore uses the shared Go-style timer heap without a native mutex. The
// scheduler calls wasmPollTimers at cooperative scheduling points and
// wasmTimerWait when it has no runnable G.

var (
	timerSchedulerHeap []*timerState
	timerSchedulerMap  map[*runtimeTimer]*timerState
)

//go:linkname registerWasmTimerHooks github.com/xgo-dev/llgo/runtime/internal/runtime.RegisterWasmTimerHooks
func registerWasmTimerHooks(poll func(), wait func() (uint64, bool))

func init() {
	registerWasmTimerHooks(wasmPollTimers, wasmTimerWait)
}

func ensureTimerScheduler() {
	if timerSchedulerMap == nil {
		timerSchedulerMap = make(map[*runtimeTimer]*timerState)
	}
}

func startRuntimeTimer(r *runtimeTimer) {
	if r == nil {
		return
	}
	ensureTimerScheduler()
	st := timerSchedulerMap[r]
	if st == nil {
		st = &timerState{r: r, heapIndex: -1}
		timerSchedulerMap[r] = st
	} else if st.active {
		timerHeapRemove(st.heapIndex)
	}
	st.callback = snapshotRuntimeTimer(r)
	st.active = true
	timerHeapAdd(st)
}

func stopRuntimeTimer(r *runtimeTimer) bool {
	if r == nil {
		return false
	}
	ensureTimerScheduler()
	st := timerSchedulerMap[r]
	wasActive := st != nil && st.active
	if wasActive {
		timerHeapRemove(st.heapIndex)
		st.active = false
		delete(timerSchedulerMap, r)
	}
	return wasActive
}

// resetRuntimeTimer updates all fields before taking the new callback snapshot.
// update is used only by the pre-Go 1.23 modTimer ABI.
func resetRuntimeTimer(r *runtimeTimer, when, period int64, update func()) bool {
	if r == nil {
		return false
	}
	ensureTimerScheduler()
	st := timerSchedulerMap[r]
	wasActive := st != nil && st.active
	if st == nil {
		st = &timerState{r: r, heapIndex: -1}
		timerSchedulerMap[r] = st
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
	return wasActive
}

func wasmTimerWait() (wait uint64, active bool) {
	if len(timerSchedulerHeap) == 0 {
		return 0, false
	}
	when := timerSchedulerHeap[0].r.when
	now := runtimeNano()
	if when <= now {
		return 0, true
	}
	return uint64(when - now), true
}

func wasmPollTimers() {
	now := runtimeNano()
	for len(timerSchedulerHeap) != 0 {
		st := timerSchedulerHeap[0]
		when := st.r.when
		if when > now {
			return
		}

		period := st.r.period
		callback := st.callback
		timerHeapRemove(0)
		if period > 0 {
			st.r.when = timerNextWhen(when, period, now)
			timerHeapAdd(st)
		} else {
			st.active = false
			delete(timerSchedulerMap, st.r)
		}

		delay := now - when
		if delay < 0 {
			delay = 0
		}
		callback.run(delay)
		now = runtimeNano()
	}
}
