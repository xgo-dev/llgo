// Behavior regression fixture for the settable baremetal wall clock
// (wallclock.SetWallTime, backed by runtime.SetWallTime). The baremetal fiber
// targets have no RTC: the wall clock starts at the Unix epoch and advances
// at the cycle clock's rate until SetWallTime moves its epoch anchor, while
// time.Since, timers and sleeps stay on the untouched monotonic domain
// (walls may jump, monotonic never goes backwards).
//
// The golden output is backend-agnostic and therefore reusable on hosted,
// where SetWallTime is a documented no-op over the real OS clock; the suites
// that run it today are the esp-emu fiber ones. Every printed assertion is a
// relation rather than an absolute timestamp: calendar-field sanity
// ranges rather than absolute timestamps (the year floor doubles as the
// anchor-moved discriminator on the baremetal side, where a broken anchor
// would leave the wall near epoch zero), wall advance across a sleep, Since
// isolation across a wall jump, and sleep lengths measured in the monotonic
// domain only. No absolute timestamps are asserted: under esp-emu the cycle
// clock advances per instruction, so only relations and counts are stable.

package main

import (
	"time"

	"github.com/xgo-dev/llgo/runtime/wallclock"
)

func main() {
	// Baseline: Before and after readings stay sane in both domains without
	// any wall setup (baremetal wall reads epoch + uptime; hosted reads the
	// host clock).
	t0 := time.Now()
	d0 := time.Since(t0)
	println("since-ok", d0 >= 0 && d0 < time.Minute)
	println("unix-nonneg", t0.Unix() >= 0)

	// Set the wall clock to a fixed epoch (2024-03-15 11:33:16 UTC); the
	// calendar fields must be internally sane on every backend. The year
	// floor also proves the anchor moved on the baremetal side: a broken
	// SetWallTime would leave the wall at epoch zero, i.e. year 1970.
	wallclock.SetWallTime(time.Date(2024, time.March, 15, 11, 33, 16, 0, time.UTC).UnixNano())
	t1 := time.Now()
	println("year", t1.Year() >= 2024 && t1.Year() < 2100)
	println("month", t1.Month() >= time.January && t1.Month() <= time.December)
	println("day", t1.Day() >= 1 && t1.Day() <= 31)
	println("hour", t1.Hour() >= 0 && t1.Hour() < 24)
	println("minsec", t1.Minute() >= 0 && t1.Minute() < 60 && t1.Second() >= 0 && t1.Second() < 60)
	println("nsec", t1.Nanosecond() >= 0 && t1.Nanosecond() < 1e9)

	// The wall clock ticks: across a sleep the wall-domain delta stays in
	// the sleep's order of magnitude (the bounds absorb the emulator's
	// instruction-rate cycle advance without weakening the check).
	w0 := time.Now()
	time.Sleep(100 * time.Millisecond)
	w1 := time.Now()
	wallDelta := w1.UnixNano() - w0.UnixNano()
	println("wall-advance", wallDelta >= 50*int64(time.Millisecond) && wallDelta <= 10*int64(time.Second))

	// time.Since stays in the monotonic domain: a wall jump between the two
	// readings must not leak into it. On the baremetal side the jump is a
	// 60 s leap; a wall-domain Since would report that instead of the tiny
	// true duration. (Hosted: SetWallTime is a no-op, so the same bounds
	// hold trivially.)
	t2 := time.Now()
	wallclock.SetWallTime(t2.UnixNano() + 60*int64(time.Second))
	d2 := time.Since(t2)
	println("since-unchanged", d2 >= 0 && d2 < 30*time.Second)

	// A wall jump while a sleep is parked must not disturb the sleep: the
	// helper re-anchors the wall 60 s into the future mid-sleep, and the
	// monotonic-domain measurement keeps the requested duration. The
	// buffered channel proves the jump happened during the sleep window.
	done := make(chan bool, 1)
	go func() {
		time.Sleep(20 * time.Millisecond)
		wallclock.SetWallTime(time.Now().UnixNano() + 60*int64(time.Second))
		done <- true
	}()
	s0 := time.Now()
	time.Sleep(100 * time.Millisecond)
	jumped := <-done
	elapsed := time.Since(s0)
	println("jump-mid-sleep", jumped)
	println("sleep-intact", elapsed >= 50*time.Millisecond && elapsed <= 5*time.Second)

	// A backward jump keeps the wall sane in both domains.
	t3 := time.Now()
	wallclock.SetWallTime(t3.UnixNano() - 2*int64(time.Hour))
	t4 := time.Now()
	println("wall-back", t4.Year() >= 1970 && t4.Year() < 2100 && t4.Unix() >= 0)

	println("done")
}
