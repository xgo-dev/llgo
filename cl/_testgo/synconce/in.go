// Behavior regression fixture: concurrent sync.Once callers contend while
// the winner still runs its function, so losers park on the Once mutex and
// must observe the completed initialization; an uncontended second Once runs
// exactly once as well. The golden output is scheduling-order independent.
package main

import (
	"runtime"
	"sync"
)

func main() {
	const workers = 6
	var once sync.Once
	calls := 0
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			once.Do(func() {
				calls++
				// Yield while initialization is in flight so another caller
				// queues on the Once mutex instead of seeing done already.
				runtime.Gosched()
			})
		}()
	}
	runtime.Gosched()
	wg.Wait()
	if calls != 1 {
		println("once ran", calls, "times")
		return
	}

	var second sync.Once
	ran := 0
	second.Do(func() { ran++ })
	second.Do(func() { ran++ })
	println("once", calls, ran)
}
