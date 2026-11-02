//go:build go1.23 && !baremetal

package runtime

import (
	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
	"unsafe"
)

// runtimeTimer is internal timer state for llgo. It is not required to match
// the Go runtime timer layout for go1.23+.
type runtimeTimer struct {
	pp       uintptr
	when     int64
	period   int64
	f        func(any, uintptr, int64)
	arg      any
	seq      uintptr
	nextwhen int64
	status   uint32
	fake     *rt.SynctestTimer
}

// timeTimer matches the beginning of time.Timer/time.Ticker for go1.23+.
// The runtime stores additional state after the first two fields.
type timeTimer struct {
	c    unsafe.Pointer
	init bool
	r    runtimeTimer
}

type runtimeTimerCallback struct {
	f   func(any, uintptr, int64)
	arg any
	seq uintptr
}

func snapshotRuntimeTimer(r *runtimeTimer) runtimeTimerCallback {
	return runtimeTimerCallback{f: r.f, arg: r.arg, seq: r.seq}
}

func (callback runtimeTimerCallback) run(delay int64) {
	if callback.f != nil {
		callback.f(callback.arg, callback.seq, delay)
	}
}

func timeSleepWake(arg any, _ uintptr, _ int64) {
	arg.(chan struct{}) <- struct{}{}
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
	return resetRuntimeTimer(&t.r, when, period, nil)
}
