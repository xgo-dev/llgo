// Behavior regression fixture for time.Ticker on the RV32 baremetal fiber
// scheduler: a 4 ms ticker delivers three ticks in order, then Stop removes
// its deadline so the timeout arm of a select wins instead of a fourth tick.
// The golden output stays identical on hosted and under esp-emu: only tick
// order and counts are printed, no absolute time. The receiver needs far
// fewer cycles between two ticks than the 4 ms period provides (emulated
// cycles advance with executed instructions), so every period delivers
// exactly one tick — the buffered chan(1) plus the skip-missed re-arm formula
// make a slow receiver lose ticks rather than queue a burst.

package main

import "time"

func main() {
	ticker := time.NewTicker(4 * time.Millisecond)
	ticks := 0
	for i := 1; i <= 3; i++ {
		<-ticker.C
		ticks++
		println("tick", ticks)
	}
	ticker.Stop()

	// Stop removed the next deadline, so no fourth tick can arrive: the
	// timeout arm wins this select.
	select {
	case <-ticker.C:
		println("unexpected tick after stop")
	case <-time.After(10 * time.Millisecond):
		println("quiet after stop")
	}
	println("done")
}
