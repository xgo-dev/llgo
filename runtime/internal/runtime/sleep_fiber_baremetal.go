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

// Scheduler sleeper list for time.Sleep on the RV32 fiber-scheduler
// baremetal targets. Sleep records the goroutine with a cycle-domain
// deadline on its own stack (parked stacks are collector roots through the
// StackRootsHook), parks, and the scheduler loop wakes expired sleepers
// whenever it picks work. Deadlines compare wrap-safely in the 32-bit cycle
// domain: int32(deadline-now) <= 0 reads as "due", exact for deadlines up
// to half a wrap window (maxSleepCycles) late. Each duration saturates at
// maxSleepCycles, but the comparison never relies on pairwise deadline
// spread — a sleeper that sat expired through a scheduler stall can share
// the list with a deadline still far ahead — so waking scans the whole
// list and reclaims every due node. Lateness beyond half a window is only
// reachable when no clock read runs anywhere for that long; it aliases
// into the future once and self-heals after a further full wrap.

// One parked Sleep. Records live on the sleeping goroutine's own frame.
type baremetalSleeper struct {
	gp       *g
	deadline uint32
	next     *baremetalSleeper
}

// Deadlines ordered by wrap-safe comparison; the earliest one heads the
// list. Single hart: the running G inserts, the bootstrap scheduler context
// removes — the two never interleave.
var baremetalSleepers *baremetalSleeper

// maxSleepCycles caps one Sleep at half a 32-bit wrap window: a deadline
// further out would alias an already-due one under the int32 comparison.
const maxSleepCycles = 1<<31 - 1

// maxSleepNs is maxSleepCycles expressed in nanoseconds at the nominal core
// frequency, the inverse of the nanotime conversion cycles*1000/MHz (~8.9 s
// at 240 MHz, ~13.4 s at 160 MHz).
var maxSleepNs = int64(maxSleepCycles) * 1000 / int64(baremetalCpuMHz)

// BaremetalSleep parks the current G for at least ns, silently clamping
// durations longer than maxSleepNs to that bound (a duration beyond half a
// wrap window cannot be expressed in the 32-bit deadline domain; TODO: loop
// segment sleeps if durations that long ever matter). The final stretch is
// a scheduler busy-poll: the single hart has no timer interrupt to sleep
// on, so RunMain spins between deadline checks. With other goroutines
// runnable they interleave normally; a lone sleeping G genuinely burns
// cycles until its deadline.
func BaremetalSleep(ns int64) {
	if ns <= 0 {
		return
	}
	s := baremetalSleeper{gp: getg(), deadline: baremetalSleepDeadline(ns)}
	baremetalInsertSleeper(&s)
	gopark()
	// Resumed by baremetalWakeSleepers, which has already unlinked the
	// expired record; the frame copy is dead from here on.
}

// baremetalSleepDeadline converts ns into a cycle-domain deadline,
// saturating at maxSleepCycles.
func baremetalSleepDeadline(ns int64) uint32 {
	if ns > maxSleepNs {
		ns = maxSleepNs
	}
	// cycles = ns * MHz / 1000, the inverse of BaremetalNanoTime's
	// cycles * 1000 / MHz. The product fits: ns is at most maxSleepNs,
	// defined so this conversion saturates exactly at maxSleepCycles.
	return riscv32Cycle() + uint32(uint64(ns)*uint64(baremetalCpuMHz)/1000)
}

// baremetalInsertSleeper links s into the deadline-ordered list, keeping the
// earliest deadline at the head as a locality hint for baremetalSleepWait.
// Ordering is an optimization only: waking never trusts the order (see
// baremetalWakeSleepers).
func baremetalInsertSleeper(s *baremetalSleeper) {
	var prev *baremetalSleeper
	for cur := baremetalSleepers; cur != nil && int32(s.deadline-cur.deadline) > 0; cur = cur.next {
		prev = cur
	}
	if prev == nil {
		s.next = baremetalSleepers
		baremetalSleepers = s
		return
	}
	s.next = prev.next
	prev.next = s
}

// baremetalWakeSleepers unlinks and readies every due sleeper, reporting
// whether it woke any. The scan covers the whole list because the int32 due
// comparison only bounds how late a deadline can read as such, not how far
// two deadlines on the list may have drifted apart while the scheduler was
// busy elsewhere; a nil list answers without touching the CSR so targets
// whose clock never runs still schedule normally. The raw reading also
// feeds the 64-bit extension, keeping it fresh across long sleeps.
func baremetalWakeSleepers() bool {
	if baremetalSleepers == nil {
		return false
	}
	now := riscv32Cycle()
	baremetalExtend(now)
	woken := false
	var prev *baremetalSleeper
	for s := baremetalSleepers; s != nil; {
		next := s.next
		if int32(s.deadline-now) <= 0 {
			if prev == nil {
				baremetalSleepers = next
			} else {
				prev.next = next
			}
			s.next = nil
			goready(s.gp)
			woken = true
		} else {
			prev = s
		}
		s = next
	}
	return woken
}

// baremetalSleepWait holds the scheduler until the earliest sleeper comes
// due, reporting whether one was woken. It returns false with no sleepers so
// the caller keeps its deadlock diagnosis. The wait is a plain spin: RV32
// baremetal has no timer interrupt to park the hart on, and WFI would sleep
// until an interrupt that never arrives.
func baremetalSleepWait() bool {
	for baremetalSleepers != nil {
		if baremetalWakeSleepers() {
			return true
		}
	}
	return false
}
