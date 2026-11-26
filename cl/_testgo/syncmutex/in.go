// Behavior regression fixture: goroutines contend for a mutex that is held
// across a scheduling yield, so waiters park in the runtime semaphore and
// resume on Unlock, and WaitGroup.Wait parks until every worker has called
// Done. The golden output is scheduling-order independent on every backend.
package main

import (
	"runtime"
	"sync"
)

func main() {
	const workers = 6
	var mu sync.Mutex
	var wg sync.WaitGroup
	counter := 0
	inside := 0

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			mu.Lock()
			// Yield while holding the lock: the next contender must park on
			// the semaphore instead of taking the fast path.
			runtime.Gosched()
			inside++
			if inside != 1 {
				println("mutex exclusion broken:", inside)
			}
			inside--
			counter++
			mu.Unlock()
		}()
	}
	// Give at least one worker the lock before waiting, so Wait observes a
	// nonzero counter and parks on the WaitGroup semaphore.
	runtime.Gosched()
	wg.Wait()
	println("mutex", counter)
}
