//go:build wasm && go1.23

package runtime

import (
	"unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
	ct "github.com/goplus/llgo/runtime/internal/clite/time"
)

type runtimeTimer struct {
	pp       uintptr
	when     int64
	period   int64
	f        func(any, uintptr, int64)
	arg      any
	seq      uintptr
	nextwhen int64
	status   uint32
	platform runtimeTimerPlatform
}

type timeTimer struct {
	c    unsafe.Pointer
	init bool
	r    runtimeTimer
}

//go:linkname time_now time.now
func time_now() (sec int64, nsec int32, mono int64) {
	tv := (*ct.Timespec)(c.Alloca(unsafe.Sizeof(ct.Timespec{})))
	ct.ClockGettime(ct.CLOCK_REALTIME, tv)
	return int64(tv.Sec), int32(tv.Nsec), runtimeNano()
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
	sleepRuntime(ns)
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
