// Goroutine-dump fixture for the ESP baremetal early panic diagnostics: one
// goroutine parks on an unbuffered channel send while the main goroutine
// panics. The report must show the current G as [running] with its scanned
// chain, and the parked G as [waiting] with the chain preserved on its fiber
// swap frame, both through the early UART0 channel.
//
// No expect.txt: the report carries build-specific PCs and pointer-derived
// goroutine ids, so the assertion lives in cl/embed_earlypanic_test.go.

package main

import "runtime"

func park() {
	ch := make(chan int)
	ch <- 1
}

func main() {
	go park()
	// Hand the scheduler a few turns so the goroutine actually runs and
	// parks on the channel send (the fiber scheduler is cooperative; the
	// parked state is what the dump reports).
	for i := 0; i < 10; i++ {
		runtime.Gosched()
	}
	var p *int
	_ = *p
}
