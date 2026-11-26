// Behavior regression fixture: mutex contention whose critical sections run
// a counted busy-wait (~2 ms of nominal core time) long enough to push a
// parked waiter past the 1 ms starvation threshold, so the release path runs
// with handoff=true and the runtime semaphore must yield the hart to the
// woken waiter instead of reacquiring around it. The golden stays an
// order-independent aggregate — mutual exclusion inside every critical
// section plus the joined counter — so it passes with or without the
// starvation mode engaging; engaging it is structural coverage, not an
// output contract.

package main

import (
	"runtime"
	"sync"
)

// burn spins through a xorshift chain (data-dependent, no closed form for
// the optimizer) for a few hundred thousand iterations — roughly 8 ms of
// nominal RV32 core time. The final value feeds an impossible branch so the
// loop cannot be folded away.
func burn(iterations int) {
	x := uint32(0x9e3779b9)
	for i := 0; i < iterations; i++ {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
	}
	if x == 0x12345678 {
		println("impossible")
	}
}

func main() {
	const workers = 3
	const rounds = 2
	var mu sync.Mutex
	var wg sync.WaitGroup
	inside := 0
	counter := 0

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for r := 0; r < rounds; r++ {
				mu.Lock()
				inside++
				if inside != 1 {
					println("mutex exclusion broken")
				}
				// Yield while holding the lock, THEN keep holding across the
				// busy-wait: the waiters queue at the yield and stay parked
				// through the whole hold, long past the 1 ms starvation
				// threshold, so their next acquire enters starving mode and
				// every release in that mode hands the token over with
				// handoff=true.
				runtime.Gosched()
				burn(500000)
				inside--
				counter++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	println("mutex", counter)
}
