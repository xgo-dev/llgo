//go:build !go1.23 && !baremetal

package runtime

import rt "github.com/xgo-dev/llgo/runtime/internal/runtime"

// runtimeTimer must match time.runtimeTimer (and runtime.timer) layout.
type runtimeTimer struct {
	pp       uintptr
	when     int64
	period   int64
	f        func(any, uintptr)
	arg      any
	seq      uintptr
	nextwhen int64
	status   uint32
	fake     *rt.SynctestTimer
}

type runtimeTimerCallback struct {
	f   func(any, uintptr)
	arg any
	seq uintptr
}

func snapshotRuntimeTimer(r *runtimeTimer) runtimeTimerCallback {
	return runtimeTimerCallback{f: r.f, arg: r.arg, seq: r.seq}
}

func (callback runtimeTimerCallback) run(_ int64) {
	if callback.f != nil {
		callback.f(callback.arg, callback.seq)
	}
}

func startTimer(r *runtimeTimer) {
	startRuntimeTimer(r)
}

func stopTimer(r *runtimeTimer) bool {
	return stopRuntimeTimer(r)
}

func resetTimer(r *runtimeTimer, when int64) bool {
	return resetRuntimeTimer(r, when, r.period, nil)
}

func modTimer(r *runtimeTimer, when, period int64, f func(any, uintptr), arg any, seq uintptr) {
	resetRuntimeTimer(r, when, period, func() {
		r.f = f
		r.arg = arg
		r.seq = seq
	})
}

func timeSleepWake(arg any, _ uintptr) {
	arg.(chan struct{}) <- struct{}{}
}
