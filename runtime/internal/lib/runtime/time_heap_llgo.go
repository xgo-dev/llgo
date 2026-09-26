//go:build !baremetal && (!wasm || (wasip1 && llgo.wasi_threads))

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

package runtime

import (
	_ "unsafe"

	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
)

// This locally ports only the part of the Go runtime timer implementation that
// does not depend on the scheduler's private M/P, netpoll, and gopark ABIs. It
// deliberately stays in LLGo's replacement runtime instead of source-patching
// or additively compiling the upstream runtime package. Like runtime/time.go,
// it keeps timers in a 4-ary heap and advances periodic timers from their
// intended deadline instead of from callback completion.

// Bound one native wait so a deadline saturated at maxTimerWhen does not
// overflow the platform timespec conversion.
const maxTimerCondWait = int64(24 * 60 * 60 * 1e9)

var (
	timerSchedulerOnce psync.Once
	timerSchedulerMu   psync.Mutex
	timerSchedulerCond psync.Cond
	timerSchedulerHeap []*timerState
	timerSchedulerMap  map[*runtimeTimer]*timerState
)

func initTimerScheduler() {
	timerSchedulerMu.Init(nil)
	initTimerSchedulerCond()
	timerSchedulerMap = make(map[*runtimeTimer]*timerState)
	go timerSchedulerLoop()
}

func ensureTimerScheduler() {
	timerSchedulerOnce.Do(initTimerScheduler)
}

// timerSchedulerHeadLocked returns a value snapshot of the current head. The
// deadline must be copied separately because reset can mutate the head timer in
// place while timerSchedulerMu is held.
func timerSchedulerHeadLocked() (*timerState, int64) {
	if len(timerSchedulerHeap) == 0 {
		return nil, 0
	}
	st := timerSchedulerHeap[0]
	return st, st.r.when
}

func signalTimerSchedulerIfHeadChangedLocked(old *timerState, oldWhen int64) {
	current, currentWhen := timerSchedulerHeadLocked()
	if current != old || currentWhen != oldWhen {
		timerSchedulerCond.Signal()
	}
}

func startRuntimeTimer(r *runtimeTimer) {
	if r == nil {
		return
	}
	ensureTimerScheduler()
	timerSchedulerMu.Lock()
	oldHead, oldHeadWhen := timerSchedulerHeadLocked()
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
	signalTimerSchedulerIfHeadChangedLocked(oldHead, oldHeadWhen)
	timerSchedulerMu.Unlock()
}

func stopRuntimeTimer(r *runtimeTimer) bool {
	if r == nil {
		return false
	}
	ensureTimerScheduler()
	timerSchedulerMu.Lock()
	oldHead, oldHeadWhen := timerSchedulerHeadLocked()
	st := timerSchedulerMap[r]
	wasActive := st != nil && st.active
	if wasActive {
		timerHeapRemove(st.heapIndex)
		st.active = false
		delete(timerSchedulerMap, r)
		signalTimerSchedulerIfHeadChangedLocked(oldHead, oldHeadWhen)
	}
	timerSchedulerMu.Unlock()
	return wasActive
}

// resetRuntimeTimer updates all fields while holding the scheduler lock.
// update is used only by the pre-Go 1.23 modTimer ABI, whose callback can also
// change during a reset.
func resetRuntimeTimer(r *runtimeTimer, when, period int64, update func()) bool {
	if r == nil {
		return false
	}
	ensureTimerScheduler()
	timerSchedulerMu.Lock()
	oldHead, oldHeadWhen := timerSchedulerHeadLocked()
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
	signalTimerSchedulerIfHeadChangedLocked(oldHead, oldHeadWhen)
	timerSchedulerMu.Unlock()
	return wasActive
}

func timerSchedulerLoop() {
	markTimerSystemG()
	timerSchedulerMu.Lock()
	for {
		// The WASI threaded collector needs the timer pthread to reach a Go
		// safepoint even when no timers are due. Poll with the lock released so
		// another thread waiting for this mutex can stop for collection too.
		if timerGCWaitQuantum != 0 {
			timerSchedulerMu.Unlock()
			timerGCSafepoint()
			timerSchedulerMu.Lock()
		}
		if len(timerSchedulerHeap) == 0 {
			if timerGCWaitQuantum == 0 {
				timerSchedulerCond.Wait(&timerSchedulerMu)
			} else {
				timerSchedulerTimedWait(timerGCWaitQuantum)
			}
			continue
		}

		st := timerSchedulerHeap[0]
		now := runtimeNano()
		if wait := timerSchedulerWaitDuration(st.r.when, now); wait > 0 {
			if timerGCWaitQuantum != 0 && wait > timerGCWaitQuantum {
				wait = timerGCWaitQuantum
			}
			timerSchedulerTimedWait(wait)
			continue
		}

		when := st.r.when
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
		timerSchedulerMu.Unlock()
		callback.run(delay)
		timerSchedulerMu.Lock()
	}
}

func timerSchedulerWaitDuration(when, now int64) int64 {
	if when <= now {
		return 0
	}
	wait := when - now
	if wait <= 0 || wait > maxTimerCondWait {
		return maxTimerCondWait
	}
	return wait
}
