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
// the CSR (see the comment on runtimeNano below).
//
// Callback timers (time.After/NewTimer/NewTicker/AfterFunc) register on the
// scheduler's deadline list when the target has a live clock
// (BaremetalCycleClock: esp32s31); the scheduler loop delivers every due
// callback on its own goroutine, the delivery context stdlib documents for
// AfterFunc. Targets whose emulated regression path cannot execute the CSR
// (esp32c3 and generic riscv32 — esp-emu models no clock for them) keep the
// degenerate stub below: runtimeNano is constant zero there, so no list
// could ever advance, only a timer already overdue at creation fires (and
// only synchronously on the creating G), and live time.After/Tick/Ticker
// timers never fire. TODO: read BaremetalNanoTime on esp32c3 too once
// esp-emu models its clock, which unlocks the timer list there as well.
//
// The stub boolean results only report whether the record exists, because
// the stub never delivers a pending timer: stopTimer/resetTimer cannot stop
// or reschedule anything that is not already overdue. On live-clock targets
// they follow the stdlib contract instead, keyed on list membership rather
// than deadline passage: true means the record was still linked (whether or
// not its deadline has passed — processing happens at the scheduler's wake
// pass, not at the deadline) and was stopped or rescheduled; false means
// the wake pass already unlinked it and queued the delivery goroutine, and
// that queued callback still runs (a reset then re-registers the timer).

// runtimeTimer is internal timer state for the fiber targets. It is not
// required to match the Go runtime timer layout for go1.23+.
type runtimeTimer struct {
	pp       uintptr
	when     int64
	period   int64
	f        func(any, uintptr, int64)
	arg      any
	seq      uintptr
	nextwhen int64
	status   uint32
	// timer is the scheduler-side record on live-clock targets; nil on the
	// stub path. The record owns the deadline from here on — the scheduler
	// unlinks and reinserts it as deadlines come due.
	timer *llruntime.BaremetalTimer
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
	if !llruntime.BaremetalCycleClock {
		// Clock-less stub: only a deadline already passed at creation fires,
		// synchronously on the creating G (where the stdlib would deliver it
		// from an independent timer goroutine). Good enough for the
		// degenerate case, wrong for live timers — documented above.
		if r.period == 0 && r.when <= runtimeNano() {
			r.f(r.arg, r.seq, runtimeNano())
		}
		return
	}
	r.timer = llruntime.BaremetalNewTimer(r.when, r.period, r.f, r.arg, r.seq)
}

func stopRuntimeTimer(r *runtimeTimer) bool {
	if r == nil {
		return false
	}
	if !llruntime.BaremetalCycleClock {
		// The stub never delivers a pending timer, so nothing can be active:
		// report that the record exists, the pre-timer-list behavior.
		return true
	}
	if r.timer == nil {
		return false
	}
	return llruntime.BaremetalStopTimer(r.timer)
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
	if !llruntime.BaremetalCycleClock {
		startRuntimeTimer(r)
		return true
	}
	// Reset forwards only when/period: the record keeps its creation-time
	// f/arg/seq. The Go 1.23+ stdlib Reset hook never changes them, and the
	// scheduler side relies on that — a callback already dispatched for the
	// old deadline reads its f/arg/seq snapshot off the record, so a future
	// divergence that mutates them here must re-create the record instead of
	// updating it in place.
	if r.timer == nil {
		r.timer = llruntime.BaremetalNewTimer(when, period, f, arg, seq)
		return false
	}
	return llruntime.BaremetalResetTimer(r.timer, when, period)
}

//go:linkname time_now time.now
func time_now() (sec int64, nsec int32, mono int64) {
	if !llruntime.BaremetalCycleClock {
		// Clock-less chip gate (esp32c3, generic riscv32): esp-emu's models
		// implement no clock, so time.Now keeps the pre-wall behavior — the
		// zero instant with no monotonic reading — and SetWallTime stays a
		// no-op.
		return 0, 0, 0
	}
	// Baremetal has no RTC: the wall clock starts at the Unix epoch and is
	// re-anchored by SetWallTime. Wall and monotonic come from one cycle-
	// clock sample and share the cycle rate; only the anchor the wall adds
	// on top ever moves, so time.Since and every timer keep the stdlib
	// monotonic contract (walls may jump, monotonic never goes backwards)
	// while the wall segment jumps as SetWallTime commands.
	wall, mono := llruntime.BaremetalWallNow()
	sec = wall / 1e9
	nsec = int32(wall % 1e9)
	if nsec < 0 {
		nsec += 1e9
		sec--
	}
	return sec, nsec, mono
}

// SetWallTime sets the baremetal wall clock to the given Unix time in
// nanoseconds; subsequent time.Now calls read the wall segment from that
// instant onward, at the cycle clock's rate. time.Since, timers and sleeps
// stay on the monotonic domain and are unaffected. On targets whose chip
// gate keeps the cycle clock off (esp32c3, generic riscv32) time.Now carries
// no wall or monotonic reading at all, and the call is a no-op.
//
// The def-side linkname renames the symbol to runtime.SetWallTime so the
// runtime/wallclock package's pull declaration links here.
//
//go:linkname SetWallTime runtime.SetWallTime
func SetWallTime(now int64) {
	if llruntime.BaremetalCycleClock {
		llruntime.BaremetalSetWallTime(now)
	}
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

// The standard library also links directly to runtime.nanotime.
func nanotime() int64 { return runtimeNano() }
