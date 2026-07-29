//go:build llgo && js && wasm && llgo.wasm_workers

package wasmevent

import _ "unsafe"

const LLGoFiles = "_wrap/event_wasm.c"

type dueTimer struct {
	timer     *Timer
	callback  Callback
	arg       any
	scheduled int64
}

var runtimeQueue queue
var runtimeQueueLock = newRuntimeMutex()

func Reset(timer *Timer, when, period int64, callback Callback, arg any) bool {
	if timer == nil {
		return false
	}
	runtimeQueueLock.Lock()
	installEventLoop(pollRuntimeQueue, waitRuntimeQueue)
	active := runtimeQueue.reset(timer, when, period, callback, arg)
	runtimeQueueLock.Unlock()
	notifyWake()
	return active
}

func Stop(timer *Timer) bool {
	runtimeQueueLock.Lock()
	active := runtimeQueue.stop(timer)
	runtimeQueueLock.Unlock()
	if active {
		notifyWake()
	}
	return active
}

func pollRuntimeQueue() int {
	now := Now()
	ran := 0
	for {
		runtimeQueueLock.Lock()
		due, ok := popRuntimeDue(now)
		runtimeQueueLock.Unlock()
		if !ok {
			return ran
		}
		if due.callback != nil {
			due.callback(due.arg, due.timer, due.scheduled, now)
		}
		runtimeQueueLock.Lock()
		if !due.timer.active {
			due.timer.callback = nil
			due.timer.arg = nil
		}
		runtimeQueueLock.Unlock()
		ran++
	}
}

func popRuntimeDue(now int64) (due dueTimer, ok bool) {
	if len(runtimeQueue.timers) == 0 {
		return due, false
	}
	timer := runtimeQueue.timers[0]
	if timer.when > now {
		return due, false
	}
	due = dueTimer{
		timer:     timer,
		callback:  timer.callback,
		arg:       timer.arg,
		scheduled: timer.when,
	}
	period := timer.period
	runtimeQueue.remove(0)
	if period > 0 {
		next := nextPeriodicDeadline(due.scheduled, period, now)
		runtimeQueue.reset(timer, next, period, due.callback, due.arg)
	}
	return due, true
}

func waitRuntimeQueue() bool {
	for {
		if pollRuntimeQueue() != 0 {
			return true
		}
		now := Now()
		deadline, ok := NextDeadline()
		if !ok {
			return false
		}
		if deadline > now {
			hostWait(uint64(deadline - now))
		}
	}
}

// NextDeadline reports the earliest active host-event deadline.
func NextDeadline() (int64, bool) {
	runtimeQueueLock.Lock()
	deadline, ok := runtimeQueue.deadline()
	runtimeQueueLock.Unlock()
	return deadline, ok
}

func Now() int64 {
	return hostNow()
}

//go:linkname hostNow C.llgo_wasm_event_now
func hostNow() int64

//go:linkname hostWait C.llgo_wasm_event_wait
func hostWait(nanoseconds uint64)
