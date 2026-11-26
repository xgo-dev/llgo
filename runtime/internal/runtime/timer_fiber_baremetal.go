//go:build llgo && baremetal && tinygo.riscv32 && !nogc

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

import "unsafe"

// Callback timer list for time.After/NewTimer/NewTicker/AfterFunc on the
// RV32 fiber-scheduler baremetal targets, fed by the scheduler loop next to
// the sleepers (sleep_fiber_baremetal.go). Every hook runs on the single
// hart: user goroutines insert and remove records through the Baremetal*
// API (called from the time hooks in internal/lib/runtime), and the
// bootstrap scheduler context scans and dispatches due records between
// scheduling two goroutines, so the list never needs a lock.
//
// Deadlines live in the 64-bit nanosecond domain of BaremetalNanoTime — the
// wrap-extended cycle clock — so due checks and the deadline-ordered insert
// compare plain int64 values and never alias across a wrap. The insertion
// order is convenience, not an invariant — nothing reads the head: waking
// scans the whole list, because the one clock snapshot a wake pass takes
// cannot see deadlines crossed by the extension while the pass itself
// allocates and dispatches.
//
// Collector reachability: pending records hang off the package global
// baremetalTimers, and the tinygogc mark phase scans that root —
// gcMarkReachable sweeps the whole globals region from _globals_start to
// _globals_end (tinygogc/gc_link.go), every package's .data/.bss lies in
// that region, and markRoots tests each word as a heap pointer — so the
// list head, the records and their func/any fields stay reachable. A record
// being dispatched has left the list; the fire record that carries it to
// its goroutine lives on the scheduler's bootstrap stack (scanBaremetalStackRoots
// marks that whole range) and then in the callback G's startarg, both
// collector-visible.

// One pending callback timer. The time hooks in internal/lib/runtime own
// the record's identity and fields; the scheduler only sees what is
// registered here.
type BaremetalTimer struct {
	f      func(any, uintptr, int64)
	arg    any
	seq    uintptr
	when   int64 // absolute deadline in BaremetalNanoTime nanoseconds
	period int64 // >0 refires at the new deadline; 0 fires once
	next   *BaremetalTimer
}

// baremetalTimers is the deadline-ordered pending list. The order has no
// readers: the scheduler's busy wait spins through wake passes and never
// inspects the head, and both waking and stopping scan the whole list. The
// ordered insert is kept anyway — it costs one comparison loop, mirrors the
// sleeper list, and leaves the earliest deadline at the head for a future
// min-deadline wait.
var baremetalTimers *BaremetalTimer

// BaremetalNewTimer registers a callback timer due at the absolute
// nanosecond deadline when, refiring every period nanoseconds while
// period > 0. The scheduler delivers each due callback on its own
// goroutine.
func BaremetalNewTimer(when, period int64, f func(any, uintptr, int64), arg any, seq uintptr) *BaremetalTimer {
	t := &BaremetalTimer{f: f, arg: arg, seq: seq, when: when, period: period}
	baremetalInsertTimer(t)
	return t
}

// BaremetalStopTimer unlinks t if it is still pending and reports whether it
// was — the stdlib stopTimer contract. Delivery has three windows: a timer
// before its deadline and a timer past its deadline that no wake pass has
// processed yet are both still linked, so Stop removes them and reports
// true — the callback never runs. Only a timer whose wake pass already
// unlinked it and queued the delivery goroutine reports false, and that
// queued callback still runs. (The stdlib collapses the first two windows:
// there, expiry and unlink are one step.)
func BaremetalStopTimer(t *BaremetalTimer) bool {
	if t == nil {
		return false
	}
	return baremetalUnlinkTimer(t)
}

// BaremetalResetTimer reschedules t at the new deadline and reports whether
// t had still been pending — the stdlib resetTimer contract, with the same
// three windows as BaremetalStopTimer: a still-linked timer (due or not)
// reschedules in place and reports true; a timer whose delivery goroutine is
// already queued reports false and is re-registered, so the queued callback
// for the old deadline still runs afterwards — the documented AfterFunc
// behavior ("Reset ... schedules f to run again").
func BaremetalResetTimer(t *BaremetalTimer, when, period int64) bool {
	if t == nil {
		return false
	}
	wasActive := baremetalUnlinkTimer(t)
	t.when = when
	t.period = period
	baremetalInsertTimer(t)
	return wasActive
}

// baremetalInsertTimer links t into deadline order, keeping the earliest
// deadline at the head and equal deadlines in registration order. Ordering
// is a convenience only: waking never trusts it (see baremetalWakeTimers).
func baremetalInsertTimer(t *BaremetalTimer) {
	var prev *BaremetalTimer
	for cur := baremetalTimers; cur != nil && t.when >= cur.when; cur = cur.next {
		prev = cur
	}
	if prev == nil {
		t.next = baremetalTimers
		baremetalTimers = t
		return
	}
	t.next = prev.next
	prev.next = t
}

// baremetalUnlinkTimer removes t from the pending list, reporting whether it
// was there. Identity search over the whole list, like the wake scan: the
// ordered position carries no contract to rely on.
func baremetalUnlinkTimer(t *BaremetalTimer) bool {
	var prev *BaremetalTimer
	for cur := baremetalTimers; cur != nil; cur = cur.next {
		if cur == t {
			if prev == nil {
				baremetalTimers = cur.next
			} else {
				prev.next = cur.next
			}
			t.next = nil
			return true
		}
		prev = cur
	}
	return false
}

// baremetalTimerNextWhen schedules the next refire of a period timer,
// stepping one period past now and dropping whole periods the delivery fell
// behind — a receiver that cannot keep up loses stale ticks instead of
// receiving a burst of late callbacks (the hosted heap's timerNextWhen).
func baremetalTimerNextWhen(when, period, now int64) int64 {
	delay := now - when
	if delay < 0 {
		delay = 0
	}
	next := when + period*(1+delay/period)
	if next < 0 {
		// The multiply overflowed into negative time; park the refire at the
		// latest expressible deadline like the hosted heap does.
		return 1<<63 - 1
	}
	return next
}

// baremetalTimerFire carries one due callback to the goroutine that runs
// it. The timer record itself cannot be the goroutine argument: resetting
// an expired one-shot reinserts the same record, so a callback already in
// flight must read an immutable snapshot instead of shared state. goexit
// frees the snapshot through releaseStartArg once the callback returned.
type baremetalTimerFire struct {
	t     *BaremetalTimer
	delta int64 // how far past the deadline the delivery happened
}

// baremetalTimerRun is the goroutine body of one timer delivery. Callbacks
// run on their own goroutine — the delivery context stdlib documents for
// AfterFunc — with the snapshot's delay so chan senders can reconstruct the
// intended send time.
func baremetalTimerRun(arg unsafe.Pointer) unsafe.Pointer {
	fire := (*baremetalTimerFire)(arg)
	fire.t.f(fire.t.arg, fire.t.seq, fire.delta)
	return nil
}

// baremetalWakeTimers unlinks and dispatches every timer due under one
// clock snapshot, reporting whether any fired. Like baremetalWakeSleepers
// it scans the whole list: dispatch allocates and spawns goroutines, and
// the allocator's collection advances the wrap extension mid-pass, so a
// node further down the list can cross its deadline while the pass is
// running and must leave on the next pass rather than trust pairwise
// deadline spread. Dispatch runs newprocBackend from the scheduler context,
// where no user goroutine can be running — single-hart cooperation keeps
// every list mutation exclusive — and the spawned G carries no creator
// (callergp nil): a timer is global state, not owned by whichever G
// happened to register it, so it inherits no bubble and no parent id.
// A nil list answers without touching the CSR, so targets whose clock never
// runs (the gated stubs never register anything) schedule normally.
func baremetalWakeTimers() bool {
	if baremetalTimers == nil {
		return false
	}
	now := BaremetalNanoTime()
	woken := false
	var prev *BaremetalTimer
	for t := baremetalTimers; t != nil; {
		next := t.next
		if t.when <= now {
			delta := now - t.when
			if prev == nil {
				baremetalTimers = next
			} else {
				prev.next = next
			}
			t.next = nil
			if t.period > 0 {
				// Refire from the original schedule, one period past now:
				// the reinserted deadline is strictly greater than this
				// pass's snapshot, so the scan below never sees it again.
				t.when = baremetalTimerNextWhen(t.when, t.period, now)
				baremetalInsertTimer(t)
			}
			fire := &baremetalTimerFire{t: t, delta: delta}
			newprocBackend(baremetalTimerRun, unsafe.Pointer(fire), goroutineStackSize, nil)
			woken = true
		} else {
			prev = t
		}
		t = next
	}
	return woken
}
