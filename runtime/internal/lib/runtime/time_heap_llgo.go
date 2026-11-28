//go:build !baremetal && (!wasm || (wasip1 && llgo.wasi_threads))

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

package runtime

import (
	"unsafe"

	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
	"github.com/xgo-dev/llgo/runtime/internal/wordmap"
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

type timerWaitState uint8

const (
	timerNotWaiting timerWaitState = iota
	timerWaitIndefinitely
	timerWaitUntil
)

var (
	timerSchedulerOnce   psync.Once
	timerSchedulerMu     psync.Mutex
	timerSchedulerCond   psync.Cond
	timerSchedulerHeap   []*timerState
	timerSchedulerFree   []*timerState
	timerSchedulerMap    wordmap.Map[unsafe.Pointer, *timerState]
	timerSchedulerWait   timerWaitState
	timerSchedulerWakeAt int64
)

const timerStateFreeCap = 64

func allocTimerStateLocked(r *runtimeTimer) *timerState {
	if n := len(timerSchedulerFree); n > 0 {
		st := timerSchedulerFree[n-1]
		timerSchedulerFree[n-1] = nil
		timerSchedulerFree = timerSchedulerFree[:n-1]
		st.r = r
		st.heapIndex = -1
		return st
	}
	return &timerState{r: r, heapIndex: -1}
}

func releaseTimerStateLocked(st *timerState) {
	// Clear references before recycling or discarding so parked schedulers
	// cannot retain timer callbacks or arguments.
	*st = timerState{heapIndex: -1}
	if len(timerSchedulerFree) < timerStateFreeCap {
		timerSchedulerFree = append(timerSchedulerFree, st)
	}
}

func initTimerScheduler() {
	initTimerSchedulerSync()
	go timerSchedulerLoop()
}

func ensureTimerScheduler() {
	timerSchedulerOnce.Do(initTimerScheduler)
}

// wakeTimerSchedulerLocked only interrupts an active wait when the scheduler
// must recheck the heap sooner. The waiter may finish a timed wait even if
// that deadline has since been removed.
func wakeTimerSchedulerLocked(when int64) {
	if timerSchedulerWait == timerWaitIndefinitely ||
		(timerSchedulerWait == timerWaitUntil && when < timerSchedulerWakeAt) {
		// A pending wake will recheck all mutations made before it reacquires
		// the lock. Avoid further notifications until the next wait begins.
		timerSchedulerWait = timerNotWaiting
		timerSchedulerSignalLocked()
	}
}

func startRuntimeTimer(r *runtimeTimer) {
	if r != nil && startSynctestTimer(r) {
		return
	}
	if r == nil {
		return
	}
	ensureTimerScheduler()
	timerSchedulerLock()
	st := timerSchedulerMap.Get(unsafe.Pointer(r))
	if st == nil {
		st = allocTimerStateLocked(r)
		timerSchedulerMap.Set(unsafe.Pointer(r), st)
	} else if st.active {
		timerHeapRemove(st.heapIndex)
	}
	st.callback = snapshotRuntimeTimer(r)
	st.active = true
	timerHeapAdd(st)
	wakeTimerSchedulerLocked(r.when)
	timerSchedulerUnlock()
}

func stopRuntimeTimer(r *runtimeTimer) bool {
	if r != nil && r.fake != nil {
		return r.fake.Stop()
	}
	if r == nil {
		return false
	}
	ensureTimerScheduler()
	timerSchedulerLock()
	st := timerSchedulerMap.Get(unsafe.Pointer(r))
	wasActive := st != nil && st.active
	if wasActive {
		timerHeapRemove(st.heapIndex)
		timerSchedulerMap.Delete(unsafe.Pointer(r))
		releaseTimerStateLocked(st)
	}
	timerSchedulerUnlock()
	return wasActive
}

// resetRuntimeTimer updates all fields while holding the scheduler lock.
// update is used only by the pre-Go 1.23 modTimer ABI, whose callback can also
// change during a reset.
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
	ensureTimerScheduler()
	timerSchedulerLock()
	st := timerSchedulerMap.Get(unsafe.Pointer(r))
	wasActive := st != nil && st.active
	if st == nil {
		st = allocTimerStateLocked(r)
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
	wakeTimerSchedulerLocked(r.when)
	timerSchedulerUnlock()
	return wasActive
}

func timerSchedulerLoop() {
	markSystemGoroutine()
	timerSchedulerLock()
	for {
		if len(timerSchedulerHeap) == 0 {
			timerSchedulerWait = timerWaitIndefinitely
			timerSchedulerWaitIndefinitely()
			timerSchedulerWait = timerNotWaiting
			continue
		}

		st := timerSchedulerHeap[0]
		now := runtimeNano()
		if wait := timerSchedulerWaitDuration(st.r.when, now); wait > 0 {
			timerSchedulerWait = timerWaitUntil
			timerSchedulerWakeAt = now + wait
			timerSchedulerWaitTimed(wait)
			timerSchedulerWait = timerNotWaiting
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
			timerSchedulerMap.Delete(unsafe.Pointer(st.r))
			releaseTimerStateLocked(st)
		}

		delay := now - when
		if delay < 0 {
			delay = 0
		}
		timerSchedulerUnlock()
		callback.run(delay)
		timerSchedulerLock()
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
