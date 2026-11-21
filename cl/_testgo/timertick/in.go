// Behavior regression fixture for time.Ticker on the RV32 baremetal fiber
// scheduler: a 5 ms ticker delivers three ticks, Stop silences it across the
// deadline that would have been the fourth tick, and Reset re-arms a stopped
// ticker at the new period. The golden output stays identical on hosted and
// under esp-emu.

package main

import "time"

func main() {
	ticker := time.NewTicker(5 * time.Millisecond)
	for i := 1; i <= 3; i++ {
		<-ticker.C
		println("tick", i)
	}
	ticker.Stop()

	// The fourth tick's deadline was removed by Stop, so the timeout arm of
	// this select must win.
	select {
	case <-ticker.C:
		println("unexpected tick after stop")
	case <-time.After(12 * time.Millisecond):
		println("quiet after stop")
	}

	ticker.Reset(6 * time.Millisecond)
	<-ticker.C
	println("tick after reset")
	ticker.Stop()
	println("done")
}
