// Behavior regression fixture: RWMutex readers hold their read lock across a
// scheduling yield so the writer only proceeds after every reader releases,
// and sync.Cond waiters park until a broadcast rechecks the predicate for
// them. Garbage collections run while waiters are parked, so the collector
// must scan the parked fibers along the way. The golden output is
// scheduling-order independent on every backend.
package main

import (
	"runtime"
	"sync"
)

func main() {
	value := 0
	var rw sync.RWMutex

	readers := 4
	var rwWg sync.WaitGroup
	rwWg.Add(readers)
	for i := 0; i < readers; i++ {
		go func() {
			defer rwWg.Done()
			rw.RLock()
			// Hold the read lock across a scheduling point so the write
			// lock can only proceed once every reader has released.
			runtime.Gosched()
			if value != 0 {
				println("reader saw dirty value:", value)
			}
			rw.RUnlock()
		}()
	}
	runtime.Gosched() // let the readers take their locks before writing
	rwWg.Wait()

	rw.Lock()
	value = 7
	rw.Unlock()

	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	done := 0
	seen := 0
	waiters := 3
	var condWg sync.WaitGroup
	condWg.Add(waiters)
	for i := 0; i < waiters; i++ {
		go func() {
			mu.Lock()
			for done == 0 {
				cond.Wait()
			}
			mu.Unlock()
			seen++
			condWg.Done()
		}()
	}
	runtime.Gosched() // give the waiters their chance to park in cond.Wait
	// Collect while the cond waiters are parked: the parked fibers and their
	// queued wait records must survive.
	runtime.GC()
	mu.Lock()
	done = 1
	cond.Broadcast()
	mu.Unlock()
	condWg.Wait()
	runtime.GC()
	println("rwcond", seen)
}
