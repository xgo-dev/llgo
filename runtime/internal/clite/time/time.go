/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package time

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const (
	LLGoPackage = "decl"
)

// -----------------------------------------------------------------------------

const (
	ClockTSize = 8
)

//go:linkname Time C.time
func Time(timer *TimeT) TimeT

//go:linkname Mktime C.mktime
func Mktime(timer *Tm) TimeT

//go:linkname Ctime C.ctime
func Ctime(timer *TimeT) string

//go:linkname Difftime C.difftime
func Difftime(end, start TimeT) float64

// -----------------------------------------------------------------------------

type Tm struct {
	Sec    c.Int
	Min    c.Int
	Hour   c.Int
	Mday   c.Int
	Mon    c.Int
	Year   c.Int
	Wday   c.Int
	Yday   c.Int
	Isdst  c.Int
	Gmtoff c.Long
	Zone   *c.Char
}

//go:linkname Gmtime C.gmtime
func Gmtime(timer *TimeT) *Tm

//go:linkname Localtime C.localtime
func Localtime(timer *TimeT) *Tm

//go:linkname Strftime C.strftime
func Strftime(buf *c.Char, bufSize uintptr, format *c.Char, timeptr *Tm) uintptr

// -----------------------------------------------------------------------------

// clock_t
type ClockT struct {
	Unused [ClockTSize]c.Char
}

//go:linkname Clock C.clock
func Clock() ClockT

// -----------------------------------------------------------------------------

type ClockidT c.Int

const (
	// the system's real time (i.e. wall time) clock, expressed as the amount of time since the Epoch.
	// This is the same as the value returned by gettimeofday
	CLOCK_REALTIME = ClockidT(0x0)

	// clock that increments monotonically, tracking the time since an arbitrary point, and will continue
	// to increment while the system is asleep.
	CLOCK_MONOTONIC = ClockidT(0x6)

	// clock that increments monotonically, tracking the time since an arbitrary point like CLOCK_MONOTONIC.
	// However, this clock is unaffected by frequency or time adjustments.  It should not be compared to
	// other system time sources.
	CLOCK_MONOTONIC_RAW = ClockidT(0x6)

	// like CLOCK_MONOTONIC_RAW, but reads a value cached by the system at context switch. This can be
	// read faster, but at a loss of accuracy as it may return values that are milliseconds old.
	// CLOCK_MONOTONIC_RAW_APPROX = ClockidT(C.CLOCK_MONOTONIC_RAW_APPROX)

	// clock that increments monotonically, in the same manner as CLOCK_MONOTONIC_RAW, but that does
	// not increment while the system is asleep. The returned value is identical to the result of
	// mach_absolute_time() after the appropriate mach_timebase conversion is applied.
	// CLOCK_UPTIME_RAW = ClockidT(C.CLOCK_UPTIME_RAW)

	// like CLOCK_UPTIME_RAW, but reads a value cached by the system at context switch. This can be read
	// faster, but at a loss of accuracy as it may return values that are milliseconds old.
	// CLOCK_UPTIME_RAW_APPROX = ClockidT(C.CLOCK_UPTIME_RAW_APPROX)

	// clock that tracks the amount of CPU (in user- or kernel-mode) used by the calling process.
	CLOCK_PROCESS_CPUTIME_ID = ClockidT(0xc)

	// clock that tracks the amount of CPU (in user- or kernel-mode) used by the calling thread.
	CLOCK_THREAD_CPUTIME_ID = ClockidT(0x10)
)

type Timespec struct {
	Sec  TimeT  // seconds
	Nsec c.Long // and nanoseconds
}

//go:linkname ClockGettime C.clock_gettime
func ClockGettime(clkId ClockidT, tp *Timespec) c.Int

//go:linkname ClockSettime C.clock_settime
func ClockSettime(clkId ClockidT, tp *Timespec) c.Int

//go:linkname ClockGetres C.clock_getres
func ClockGetres(clkId ClockidT, res *Timespec) c.Int

// -----------------------------------------------------------------------------
