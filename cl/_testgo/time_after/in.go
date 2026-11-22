// Behavior regression fixture for time.After and select's default branch on
// the RV32 baremetal fiber scheduler: two After timers with different delays
// deliver strictly in deadline order, a select with a default arm answers
// immediately while the later timer is still pending, and the same select
// falls to default again once both channels are drained. The golden output
// stays identical on hosted and under esp-emu: no absolute time is printed,
// the 5 ms gap between the two deadlines is orders of magnitude more cycles
// than the receiver needs between two wakes (emulated cycles advance with
// executed instructions, so only relative order carries meaning).

package main

import "time"

func main() {
	first := time.After(4 * time.Millisecond)
	second := time.After(9 * time.Millisecond)

	// Only the 4 ms timer can be due while the 9 ms one is still pending:
	// deadline order, not creation order, decides the delivery.
	<-first
	println("after 4ms")

	// The 9 ms deadline is still far away, so the default arm must win
	// without blocking.
	select {
	case <-second:
		println("unexpected early delivery")
	default:
		println("default while pending")
	}

	<-second
	println("after 9ms")

	// Both channels are drained now: the buffered delivery slot is empty and
	// no timer is armed, so the default arm answers again.
	select {
	case <-first:
		println("unexpected stale value")
	case <-second:
		println("unexpected second value")
	default:
		println("default after drain")
	}
	println("done")
}
