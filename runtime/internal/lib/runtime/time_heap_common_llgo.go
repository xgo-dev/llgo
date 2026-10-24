//go:build !baremetal

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

package runtime

import (
	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
	_ "unsafe"
)

// This is the target-independent part of LLGo's local port of the Go runtime
// timer implementation. Native and WebAssembly schedulers share the same
// 4-ary heap and periodic-deadline rules; only their wait/wakeup backends
// differ.

const (
	timerHeapArity = 4
	maxTimerWhen   = int64(1<<63 - 1)
)

type timerState struct {
	r         *runtimeTimer
	callback  runtimeTimerCallback
	heapIndex int
	active    bool
}

func timerNextWhen(when, period, now int64) int64 {
	delay := now - when
	if delay < 0 {
		delay = 0
	}
	next := when + period*(1+delay/period)
	if next < 0 {
		next = maxTimerWhen
	}
	return next
}

//go:linkname timeSleep time.Sleep
func timeSleep(ns int64) {
	if ns <= 0 {
		return
	}
	when := synctestNano() + ns
	if when < 0 {
		when = maxTimerWhen
	}
	done := make(chan struct{}, 1)
	r := &runtimeTimer{
		when: when,
		f:    timeSleepWake,
		arg:  done,
	}
	startRuntimeTimer(r)
	<-done
}

func timerHeapLess(i, j int) bool {
	return timerSchedulerHeap[i].r.when < timerSchedulerHeap[j].r.when
}

func timerHeapSwap(i, j int) {
	timerSchedulerHeap[i], timerSchedulerHeap[j] = timerSchedulerHeap[j], timerSchedulerHeap[i]
	timerSchedulerHeap[i].heapIndex = i
	timerSchedulerHeap[j].heapIndex = j
}

func timerHeapAdd(st *timerState) {
	st.heapIndex = len(timerSchedulerHeap)
	timerSchedulerHeap = append(timerSchedulerHeap, st)
	timerHeapSiftUp(st.heapIndex)
}

func timerHeapRemove(i int) {
	n := len(timerSchedulerHeap) - 1
	removed := timerSchedulerHeap[i]
	if i != n {
		timerHeapSwap(i, n)
	}
	timerSchedulerHeap[n] = nil
	timerSchedulerHeap = timerSchedulerHeap[:n]
	removed.heapIndex = -1
	if i != n && !timerHeapSiftDown(i) {
		timerHeapSiftUp(i)
	}
}

func timerHeapSiftUp(i int) {
	for i > 0 {
		p := (i - 1) / timerHeapArity
		if !timerHeapLess(i, p) {
			return
		}
		timerHeapSwap(i, p)
		i = p
	}
}

func timerHeapSiftDown(i int) bool {
	moved := false
	for {
		child := timerHeapArity*i + 1
		if child >= len(timerSchedulerHeap) {
			return moved
		}
		best := child
		limit := child + timerHeapArity
		if limit > len(timerSchedulerHeap) {
			limit = len(timerSchedulerHeap)
		}
		for candidate := child + 1; candidate < limit; candidate++ {
			if timerHeapLess(candidate, best) {
				best = candidate
			}
		}
		if !timerHeapLess(best, i) {
			return moved
		}
		timerHeapSwap(i, best)
		i = best
		moved = true
	}
}

//go:linkname time_now time.now
func time_now() (sec int64, nsec int32, mono int64) {
	if b := rt.CurrentSynctestBubble(); b != nil {
		now := b.Now()
		// A fake monotonic epoch cannot be compared to real outside times.
		// Keep fake nanoseconds for scheduling, but omit time.Time's reading.
		return now / 1e9, int32(now % 1e9), 0
	}
	sec, nsec = walltime()
	mono = runtimeNano()
	return
}

//go:linkname time_runtimeNow time.runtimeNow
func time_runtimeNow() (sec int64, nsec int32, mono int64) {
	return time_now()
}

//go:linkname time_runtimeNano time.runtimeNano
func time_runtimeNano() int64 {
	return synctestNano()
}

//go:linkname time_runtimeIsBubbled time.runtimeIsBubbled
func time_runtimeIsBubbled() bool {
	return rt.CurrentSynctestBubble() != nil
}

func synctestNano() int64 {
	if b := rt.CurrentSynctestBubble(); b != nil {
		return b.Now()
	}
	return runtimeNano()
}

func startSynctestTimer(r *runtimeTimer) bool {
	if r.fake == nil && rt.CurrentSynctestBubble() == nil {
		return false
	}
	if r.fake != nil {
		r.fake.Check()
	}
	callback := snapshotRuntimeTimer(r)
	if r.fake == nil {
		r.fake = rt.NewSynctestTimer(r.when, r.period, callback.run)
	} else {
		r.fake.Reset(r.when, r.period, callback.run)
	}
	return true
}
