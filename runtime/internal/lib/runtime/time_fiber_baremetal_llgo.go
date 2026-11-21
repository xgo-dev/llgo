//go:build llgo && baremetal && tinygo.riscv32 && !nogc && go1.23

/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package runtime

import (
	"unsafe"

	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

// Time hooks for the RV32 fiber-scheduler baremetal targets. time.Sleep
// parks the current G on the scheduler's sleeper list until its cycle-domain
// deadline expires (sleep_fiber_baremetal.go), and runtimeNano reads the
// wrap-extended cycle clock where the target's regression path can execute
// the CSR (see the comment on runtimeNano below). The timer plumbing below
// stays minimal: a timer that has not already passed its deadline at
// creation never fires, so time.After/Tick/Ticker remain unusable.
//
// TODO: grow callback-timer delivery (a deadline-ordered timer list fed by
// the scheduler loop next to the sleepers in internal/runtime) and route
// startRuntimeTimer/resetRuntimeTimer through it.

// The timer stubs never deliver a pending timer, so their boolean results
// only report whether the record exists: stopTimer/resetTimer cannot stop
// or reschedule anything that is not already overdue. startRuntimeTimer
// runs f synchronously on the creating G when the deadline has already
// passed at creation, where the stdlib would deliver it from an independent
// timer goroutine — good enough for the already-overdue degenerate case,
// wrong for live timers (see the TODO above).
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
	// Baremetal has no RTC: the wall clock stays at the zero instant, so
	// time.Now carries no meaningful absolute time.
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
	llruntime.BaremetalSleep(ns)
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

// runtimeNano is the monotonic clock every runtime hook in this file
// derives from. Like the sema feed, it reads the cycle clock only on targets
// whose emulated regression suite can execute the CSR read: time.init and
// time.Since call it on every fixture importing time, and esp-emu's esp32c3
// model has no clock to trap into. time.Sleep stays real on all fiber
// targets — only fixtures that actually sleep touch the counter, and those
// are excluded from the C3 emulation suite.
//
// TODO: read BaremetalNanoTime on esp32c3 too once esp-emu models its clock.
func runtimeNano() int64 {
	if llruntime.BaremetalCycleClock {
		return llruntime.BaremetalNanoTime()
	}
	return 0
}
