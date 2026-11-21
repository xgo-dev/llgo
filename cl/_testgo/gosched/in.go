// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.
package main

import "runtime"

// Two goroutines ping-pong through Gosched handoffs. Only the final round
// count is printed: interleaving differs between preemptive and cooperative
// backends, so per-step output would fork.
func main() {
	a := make(chan int)
	b := make(chan int)
	done := make(chan struct{})
	const rounds = 200
	go func() {
		for i := 0; i < rounds; i++ {
			<-a
			runtime.Gosched()
			b <- i
		}
		close(done)
	}()
	go func() {
		for i := 0; i < rounds; i++ {
			a <- i
			<-b
			runtime.Gosched()
		}
	}()
	<-done
	println("ping rounds", rounds)
}
