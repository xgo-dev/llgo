//go:build wasm && go1.23 && !(wasip1 && llgo.wasi_threads)

package runtime

import "github.com/goplus/llgo/runtime/internal/wasmevent"

type runtimeTimerPlatform struct {
	event wasmevent.Timer
}

func startRuntimeTimer(r *runtimeTimer) {
	if r == nil || r.f == nil {
		return
	}
	wasmevent.Reset(&r.platform.event, r.when, r.period, fireWasmRuntimeTimer, r)
}

func stopRuntimeTimer(r *runtimeTimer) bool {
	return r != nil && wasmevent.Stop(&r.platform.event)
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
	return wasmevent.Reset(&r.platform.event, when, period, fireWasmRuntimeTimer, r)
}

func fireWasmRuntimeTimer(arg any, timer *wasmevent.Timer, scheduled, now int64) {
	r := arg.(*runtimeTimer)
	if deadline, active := timer.Deadline(); active {
		r.when = deadline
	}
	f, farg, seq := r.f, r.arg, r.seq
	delta := now - scheduled
	if delta < 0 {
		delta = 0
	}
	if f != nil {
		f(farg, seq, delta)
	}
}

func sleepRuntime(ns int64) {
	if ns <= 0 {
		return
	}
	done := make(chan struct{}, 1)
	r := &runtimeTimer{
		when: runtimeNano() + ns,
		f:    timeSleepWake,
		arg:  done,
	}
	startRuntimeTimer(r)
	<-done
	stopRuntimeTimer(r)
}

func timeSleepWake(arg any, _ uintptr, _ int64) {
	arg.(chan struct{}) <- struct{}{}
}
