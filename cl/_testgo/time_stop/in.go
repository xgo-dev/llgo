// Behavior regression fixture for time.NewTimer's Stop and Reset on the RV32
// baremetal fiber scheduler: Stop on a pending timer reports it as active and
// prevents the delivery, the channel stays empty (the buffered slot is never
// filled), and Reset re-arms the stopped timer so its value arrives. The
// golden output stays identical on hosted and under esp-emu: the first
// deadline is still several milliseconds out when Stop runs — orders of
// magnitude more cycles than the few instructions between NewTimer and Stop
// need — so "pending" is not a race, and no absolute time is printed.

package main

import "time"

func main() {
	timer := time.NewTimer(6 * time.Millisecond)

	// The timer is still pending, so Stop must report true and keep the
	// delivery from ever happening.
	if timer.Stop() {
		println("stop pending true")
	} else {
		println("stop pending false")
	}

	// Nothing was delivered: the buffered chan(1) stays empty and the timer
	// is disarmed, so the default arm answers immediately.
	select {
	case <-timer.C:
		println("unexpected delivery after stop")
	default:
		println("no delivery after stop")
	}

	// Reset re-arms the stopped timer. Both backends report it as inactive at
	// the Reset (Stop had already unlinked it).
	if timer.Reset(5 * time.Millisecond) {
		println("reset inactive true")
	} else {
		println("reset inactive false")
	}

	<-timer.C
	println("delivered after reset")
	println("done")
}
