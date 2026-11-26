// Behavior regression fixture: goroutines contend for a mutex across several
// garbage collection rounds, so the collector scans the stacks of fibers
// parked inside Lock and of fibers parked in WaitGroup.Wait, and every local
// object a parked fiber still references survives the collection. The golden
// output is scheduling-order independent on every backend.
package main

import (
	"runtime"
	"sync"
)

type record struct {
	id   int
	note [8]int
}

var (
	mu      sync.Mutex
	records []*record
)

func main() {
	const rounds = 6
	var wg sync.WaitGroup
	wg.Add(rounds)
	for i := 0; i < rounds; i++ {
		go func(n int) {
			defer wg.Done()
			// Local state stays live across the park inside Lock: while this
			// goroutine waits for the mutex, the collector must scan its
			// parked fiber stack or these objects would be collected.
			local := &record{id: n}
			for k := range local.note {
				local.note[k] = n*10 + k
			}
			mu.Lock()
			runtime.Gosched() // widen the contention window
			records = append(records, local)
			mu.Unlock()
			if local.id != n {
				println("local id lost:", local.id, n)
				return
			}
			for k := range local.note {
				if local.note[k] != n*10+k {
					println("local note lost:", local.id, k)
					return
				}
			}
		}(i)
	}
	runtime.Gosched() // let one worker hold the mutex before the GC rounds
	for i := 0; i < 4; i++ {
		runtime.GC()
		runtime.Gosched() // rotate the scheduler while workers queue on mu
	}
	wg.Wait()
	if len(records) != rounds {
		println("records lost:", len(records))
		return
	}
	for _, r := range records {
		if r.id < 0 || r.id >= rounds || r.note[0] != r.id*10 {
			println("record corrupted:", r.id)
			return
		}
	}
	println("gcfight", len(records))
}
