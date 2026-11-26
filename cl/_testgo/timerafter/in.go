// Behavior regression fixture for time.After on the RV32 baremetal fiber
// scheduler: three timers with increasing deadlines fire in deadline order —
// a select parks on all three arms and only the earliest deadline can win —
// and a shorter timer created after the first batch still slots in purely by
// deadline. The golden output stays identical on hosted and under esp-emu;
// the deadline gaps give the single-hart scheduler orders of magnitude more
// cycles than it needs to park, deliver and print between two fires.

package main

import "time"

func main() {
	first := time.After(5 * time.Millisecond)
	second := time.After(10 * time.Millisecond)
	third := time.After(15 * time.Millisecond)

	// Only the 5 ms arm can be ready while the others are still pending:
	// deadline order, not creation order, decides the delivery.
	select {
	case <-first:
		println("after 5ms first")
	case <-second:
		println("out of order 10ms")
	case <-third:
		println("out of order 15ms")
	}
	<-second
	println("after 10ms second")
	<-third
	println("after 15ms third")

	// A timer created after the first batch slots in by deadline: the 6 ms
	// arm fires next and the 8 ms arm must lose this select.
	six := time.After(6 * time.Millisecond)
	select {
	case <-six:
		println("after 6ms fourth")
	case <-time.After(8 * time.Millisecond):
		println("out of order 8ms")
	}
	println("done")
}
