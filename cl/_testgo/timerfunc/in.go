// Behavior regression fixture for time.AfterFunc on the RV32 baremetal fiber
// scheduler: Reset on a pending timer reschedules the callback (the old
// deadline must not fire), Reset on an expired timer schedules it again and
// reports the timer as inactive, and Stop after expiry reports nothing to
// stop. The callback runs on its own goroutine and signals through a
// buffered channel; the counter is unsynchronized single-hart-ordered state
// like the other timing fixtures'. The golden output stays identical on
// hosted and under esp-emu.

package main

import "time"

func main() {
	count := 0
	done := make(chan bool, 2)
	timer := time.AfterFunc(5*time.Millisecond, func() {
		count++
		done <- true
	})

	// Reset while the 5 ms deadline is still pending: the old callback must
	// never run, the rescheduled one fires exactly once.
	if timer.Reset(10 * time.Millisecond) {
		println("reset pending true")
	} else {
		println("reset pending false")
	}
	<-done
	println("single fire", count == 1)

	// The timer has expired, so Reset schedules the callback again and
	// reports the timer as inactive.
	if timer.Reset(6 * time.Millisecond) {
		println("reset expired true")
	} else {
		println("reset expired false")
	}
	<-done
	println("double fire", count == 2)

	if timer.Stop() {
		println("stop expired true")
	} else {
		println("stop expired false")
	}
	println("done")
}
