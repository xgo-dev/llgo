//go:build wasip1 && wasm && go1.23 && llgo.wasi_threads

package runtime

type runtimeTimerPlatform struct{}

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

func sleepRuntime(ns int64) {
	if ns <= 0 {
		return
	}
	deadline := runtimeNano() + ns
	for runtimeNano() < deadline {
	}
}
