// Behavior regression fixture for time.Sleep on the RV32 baremetal fiber
// scheduler: two sleep durations produce their outputs in order, and a busy
// worker goroutine advances further during the longer sleep than during the
// shorter one, so the relative duration difference is observable without
// printing any timestamp. The golden output stays identical on hosted and
// under esp-emu.

package main

import (
	"runtime"
	"time"
)

// counter accumulates worker quanta. The worker runs in fixed-size quanta
// and yields between them, so the sleeping main goroutine is scheduled back
// as soon as its sleep deadline expires.
//
// counter is deliberately unsynchronized: the single-hart baremetal
// scheduler orders every access strictly, and the fixtures never run under
// -race, so the hosted unsynchronized access is a known, accepted trade-off
// for a timing fixture.
var counter int

func worker() {
	for {
		for i := 0; i < 1000; i++ {
			counter++
		}
		runtime.Gosched()
	}
}

func main() {
	go worker()
	counter = 0
	time.Sleep(20 * time.Millisecond)
	short := counter
	println("short-progress", short > 0)
	counter = 0
	time.Sleep(60 * time.Millisecond)
	long := counter
	println("long-progress", long > 0)
	println("long-faster", long > short)
	println("done")
}
