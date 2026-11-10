//go:build llgo && !wasm && !baremetal && !nogc

package gotest

import (
	"runtime"
	"testing"
)

func TestChannelWaiterSteadyStateAllocations(t *testing.T) {
	for _, capacity := range []int{0, 1} {
		jobs := make(chan uint64, capacity)
		replies := make(chan uint64, capacity)
		done := make(chan struct{})
		go func() {
			defer close(done)
			for value := range jobs {
				replies <- value
			}
		}()
		exchange := func(n int) {
			for i := 0; i < n; i++ {
				jobs <- uint64(i)
				if value := <-replies; value != uint64(i) {
					t.Fatalf("received %d, want %d", value, i)
				}
			}
		}
		// Exclude goroutine/channel setup and lazy runtime initialization.
		exchange(2000)
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		exchange(10000)
		runtime.ReadMemStats(&after)
		close(jobs)
		<-done
		allocated := after.TotalAlloc - before.TotalAlloc
		t.Logf("capacity %d: %d collector bytes / 10000 exchanges", capacity, allocated)
		// TotalAlloc is collector accounting, not Go's allocation counter.
		// Allow incidental runtime work but reject per-exchange waiter heaps.
		// Each capacity has its own measured delta and bound. LLGo does not
		// populate MemStats.Mallocs, so it cannot provide an allocation gate.
		if allocated > 64<<10 {
			t.Errorf("capacity %d: steady-state channel exchanges allocated %d bytes", capacity, allocated)
		}
	}
}
