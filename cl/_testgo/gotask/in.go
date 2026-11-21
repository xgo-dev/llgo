// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

import "runtime"

func main() {
	println("numgo", runtime.NumGoroutine() >= 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		runtime.Gosched()
	}()
	<-done
	runtime.Gosched()
	println("sched ok")
	c := make(chan int, 1)
	go func() {
		defer func() { c <- 99 }()
		runtime.Goexit()
	}()
	println("goexit", <-c)
	println("numgo2", runtime.NumGoroutine() >= 1)
}
