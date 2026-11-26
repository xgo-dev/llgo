// Behavior regression fixture for time.Sleep across goroutines on the RV32
// baremetal fiber scheduler: three workers sleep for distinct durations and
// must report in duration order, and the main goroutine parks in
// WaitGroup.Wait until every worker finished. The golden output stays
// identical on hosted and under esp-emu.

package main

import (
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
		println("worker short")
	}()
	go func() {
		defer wg.Done()
		time.Sleep(30 * time.Millisecond)
		println("worker long")
	}()
	go func() {
		defer wg.Done()
		time.Sleep(20 * time.Millisecond)
		println("worker mid")
	}()
	wg.Wait()
	println("main joined")
}
