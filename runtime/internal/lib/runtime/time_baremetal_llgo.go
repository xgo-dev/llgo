//go:build baremetal && go1.23

package runtime

import "unsafe"

// Minimal timer hooks for baremetal builds.
// They keep stdlib packages linkable for target smoke tests but do not
// provide asynchronous timer delivery.

type runtimeTimer struct {
	pp       uintptr
	when     int64
	period   int64
	f        func(any, uintptr, int64)
	arg      any
	seq      uintptr
	nextwhen int64
	status   uint32
}

type timeTimer struct {
	c    unsafe.Pointer
	init bool
	r    runtimeTimer
}

func startRuntimeTimer(r *runtimeTimer) {
	if r == nil || r.f == nil {
		return
	}
	if r.period == 0 && r.when <= runtimeNano() {
		r.f(r.arg, r.seq, runtimeNano())
	}
}

func stopRuntimeTimer(r *runtimeTimer) bool {
	return r != nil
}

func resetRuntimeTimer(r *runtimeTimer, when, period int64, f func(any, uintptr, int64), arg any, seq uintptr) bool {
	if r == nil {
		return false
	}
	r.when = when
	r.period = period
	r.f = f
	r.arg = arg
	r.seq = seq
	startRuntimeTimer(r)
	return true
}

//go:linkname time_now time.now
func time_now() (sec int64, nsec int32, mono int64) {
	return 0, 0, 0
}

//go:linkname time_runtimeNow time.runtimeNow
func time_runtimeNow() (sec int64, nsec int32, mono int64) {
	return time_now()
}

//go:linkname time_runtimeNano time.runtimeNano
func time_runtimeNano() int64 {
	return runtimeNano()
}

//go:linkname time_runtimeIsBubbled time.runtimeIsBubbled
func time_runtimeIsBubbled() bool {
	return false
}

//go:linkname timeSleep time.Sleep
func timeSleep(ns int64) {
	_ = ns
}

//go:linkname newTimer time.newTimer
func newTimer(when, period int64, f func(any, uintptr, int64), arg any, cp unsafe.Pointer) *timeTimer {
	t := &timeTimer{c: cp, init: true}
	t.r.when = when
	t.r.period = period
	t.r.f = f
	t.r.arg = arg
	startRuntimeTimer(&t.r)
	return t
}

//go:linkname stopTimer time.stopTimer
func stopTimer(t *timeTimer) bool {
	if t == nil {
		return false
	}
	return stopRuntimeTimer(&t.r)
}

//go:linkname resetTimer time.resetTimer
func resetTimer(t *timeTimer, when, period int64) bool {
	if t == nil {
		return false
	}
	return resetRuntimeTimer(&t.r, when, period, t.r.f, t.r.arg, t.r.seq)
}

func runtimeNano() int64 {
	return 0
}
