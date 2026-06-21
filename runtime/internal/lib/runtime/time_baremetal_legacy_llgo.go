//go:build baremetal && !go1.23

package runtime

// Minimal timer hooks for baremetal builds.
// They keep stdlib packages linkable for target smoke tests but do not
// provide asynchronous timer delivery.

type runtimeTimer struct {
	pp       uintptr
	when     int64
	period   int64
	f        func(any, uintptr)
	arg      any
	seq      uintptr
	nextwhen int64
	status   uint32
}

func startTimer(r *runtimeTimer) {
	if r == nil || r.f == nil {
		return
	}
	if r.period == 0 && r.when <= runtimeNano() {
		r.f(r.arg, r.seq)
	}
}

func stopTimer(r *runtimeTimer) bool {
	return r != nil
}

func resetTimer(r *runtimeTimer, when int64) bool {
	if r == nil {
		return false
	}
	r.when = when
	startTimer(r)
	return true
}

func modTimer(r *runtimeTimer, when, period int64, f func(any, uintptr), arg any, seq uintptr) {
	if r == nil {
		return
	}
	r.when = when
	r.period = period
	r.f = f
	r.arg = arg
	r.seq = seq
	startTimer(r)
}

func resetRuntimeTimer(r *runtimeTimer, when, period int64, f func(any, uintptr), arg any, seq uintptr) bool {
	if r == nil {
		return false
	}
	modTimer(r, when, period, f, arg, seq)
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

func runtimeNano() int64 {
	return 0
}
