package sync_test

import (
	"sync"
	"testing"
)

func TestMutexSpinContentionStress(t *testing.T) {
	var mu sync.Mutex
	var counter int
	var wg sync.WaitGroup
	const workers = 8
	const iters = 5000
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				mu.Lock()
				counter++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if counter != workers*iters {
		t.Fatalf("counter = %d, want %d", counter, workers*iters)
	}
}
