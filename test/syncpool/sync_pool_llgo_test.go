//go:build llgo

package syncpool

import (
	"sync"
	"testing"
)

// These tests rely on LLGo's thread-local Pool retention and 1:1
// goroutine/thread binding to retrieve the values placed by each goroutine.

// Regression: these live pools exhausted host keys with one key per pool.
func TestManyLivePools(t *testing.T) {
	pools := make([]sync.Pool, 2048)
	for i := range pools {
		pools[i].Put(i)
	}
	for i := range pools {
		if got := pools[i].Get(); got != i {
			t.Fatalf("pool %d: got %v", i, got)
		}
	}
}

// Successive goroutines exercise cumulative pool creation across thread exits
// on LLGo's 1:1 goroutine/thread backend.
func TestManyPoolsAcrossSuccessiveThreads(t *testing.T) {
	for batch := 0; batch < 40; batch++ {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 64; i++ {
				var p sync.Pool
				p.Put(i)
				if got := p.Get(); got != i {
					t.Errorf("pool %d: got %v", i, got)
				}
			}
		}()
		wg.Wait()
	}
}
