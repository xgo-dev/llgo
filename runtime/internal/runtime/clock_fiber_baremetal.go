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

import _ "unsafe" // for go:linkname

// Monotonic clock for the RV32 fiber-scheduler baremetal targets, read from
// the RISC-V cycle CSR through _wrap/clock_riscv32.c.
//
// RV32 exposes only the low 32 bits of the counter, so every reader goes
// through a wrap-safe forward-distance extension: the modular uint32
// difference is true forward progress on this single-hart runtime (the
// counter never runs backwards between two cooperative reads), and it stays
// exact as long as consecutive reads land within one full window (2^32
// cycles, ~17.9 s at 240 MHz) of each other. baremetalWakeSleepers feeds its
// raw reading back into the extension while sleepers are parked, which is
// what keeps the bound under long sleeps: the deadline spin itself reads
// the bare CSR without refreshing it.

//go:linkname riscv32Cycle C.llgoRiscv32Cycle
func riscv32Cycle() uint32

var baremetalClock struct {
	last uint32
	base uint64 // accumulated cycles across 32-bit wraps
	full bool   // last holds a valid reading
}

// baremetalExtend folds one raw CSR reading into the wrap-extended cycle
// count and returns it.
func baremetalExtend(now uint32) uint64 {
	if !baremetalClock.full {
		baremetalClock.last = now
		baremetalClock.full = true
		return 0
	}
	baremetalClock.base += uint64(now - baremetalClock.last)
	baremetalClock.last = now
	return baremetalClock.base
}

// baremetalCycles returns the wrap-extended cycle count since the first
// clock read.
func baremetalCycles() uint64 {
	return baremetalExtend(riscv32Cycle())
}

// BaremetalNanoTime reports the monotonic clock in nanoseconds. Only
// relative durations are guaranteed: the conversion divides by the nominal
// core frequency (see z_clock_freq_*.go), which matches real silicon but not
// esp-emu's instruction-rate cycle advance, so absolute wall-clock duration
// is stretched under emulation while ordering and ratios stay exact.
//
// It feeds every monotonic consumer: time.Now's monotonic segment,
// time.Since, timer deadline arithmetic, fips mono and the mutex starvation
// feed. The wall segment of time.Now carries the same rate but its own epoch
// anchor (see baremetalWallBase); time.Sleep keeps its own cycle-domain
// deadlines and never calls this. See runtimeNano in lib/runtime for the
// per-target gate.
func BaremetalNanoTime() int64 {
	return int64(baremetalCycles() * 1000 / uint64(baremetalCpuMHz))
}

// baremetalWallBase anchors the wall clock: the Unix-nanosecond reading the
// wall domain reports when the monotonic reading is zero. Baremetal has no
// RTC, so the anchor starts at zero and every wall reading is anchor + mono
// — the wall advances at the cycle clock's rate until SetWallTime moves the
// anchor alone. A scalar package global: it sits in the .data/.bss globals
// region the conservative collector sweeps as roots (see the reachability
// note on baremetalTimers) and holds no pointer words, so nothing beyond
// static allocation is needed to keep it valid.
var baremetalWallBase int64

// BaremetalWallNow returns one consistent snapshot: wall is the wall clock
// in nanoseconds since the Unix epoch (anchor plus the current mono
// reading), mono is the wrap-extended cycle-domain reading the wall was
// derived from. Both come from a single cycle sample, so time.Now's two
// segments never straddle a SetWallTime re-anchor on the single-hart
// scheduler.
func BaremetalWallNow() (wall int64, mono int64) {
	mono = BaremetalNanoTime()
	return baremetalWallBase + mono, mono
}

// BaremetalSetWallTime re-anchors the wall clock so it reads now (Unix
// nanoseconds) at the current instant. The wall segment may jump forward or
// backward freely — the stdlib wall-clock contract — while every monotonic
// consumer (time.Since, timer deadlines, sleep deadlines, the mutex
// starvation feed) keeps reading the untouched cycle domain.
//
// Single-hart cooperative scheduler: the anchor is a plain scalar and a
// caller never yields inside the read-modify-write, so wall readings cannot
// tear here. A hypothetical SMP port would have to make the anchor atomic.
func BaremetalSetWallTime(now int64) {
	baremetalWallBase = now - BaremetalNanoTime()
}

// BaremetalSchedNanoTime feeds the mutex starvation threshold. It reads the
// cycle clock only on targets whose emulated regression suite can execute
// the CSR read (esp32s31; esp-emu's esp32c3 model has no clock, and the
// contention fixtures park and wake mutexes on every run). Elsewhere it
// stays zero, which keeps every mutex waiter on the normal (non-starving)
// path — exactly the pre-clock behavior.
//
// TODO: wire esp32c3 here once esp-emu gives its model a monotonic clock.
func BaremetalSchedNanoTime() int64 {
	if BaremetalCycleClock {
		return BaremetalNanoTime()
	}
	return 0
}
