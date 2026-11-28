package sync_test

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestPoolHybridFastAndOverflowSlots(t *testing.T) {
	// Allocate more than poolFastSlotLimit (1024) pools to exercise both the
	// O(1) fast-slot slice path and the overflow wordmap path.
	const total = 1040
	pools := make([]sync.Pool, total)
	for i := range pools {
		idx := i
		pools[i].Put(idx)
	}
	for i := range pools {
		got := pools[i].Get()
		if got != i {
			t.Fatalf("pool[%d].Get() = %v, want %d", i, got, i)
		}
	}

	// Verify TLS destructor publishes both a fast-slot pool local and an
	// overflow pool local into Pool.victim when a locked OS thread exits.
	fastPool := &pools[0]
	overflowPool := &pools[total-1]

	done := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		// Put twice per pool so the second value lands in poolLocal.shared,
		// which Pool.getSlow pops after the TLS destructor publishes victim.
		fastPool.Put("fast-private")
		fastPool.Put("from-fast-worker")
		overflowPool.Put("overflow-private")
		overflowPool.Put("from-overflow-worker")
		close(done)
	}()
	<-done

	if runtime.Compiler == "llgo" && runtime.GOARCH != "wasm" {
		deadline := time.Now().Add(500 * time.Millisecond)
		var gotFast, gotOverflow any
		for time.Now().Before(deadline) {
			if gotFast == nil {
				gotFast = fastPool.Get()
			}
			if gotOverflow == nil {
				gotOverflow = overflowPool.Get()
			}
			if gotFast != nil && gotOverflow != nil {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if gotFast != "from-fast-worker" {
			t.Fatalf("fastPool.Get() = %v, want from-fast-worker", gotFast)
		}
		if gotOverflow != "from-overflow-worker" {
			t.Fatalf("overflowPool.Get() = %v, want from-overflow-worker", gotOverflow)
		}
		return
	}

	if got := fastPool.Get(); got != nil && got != "from-fast-worker" {
		t.Fatalf("fastPool.Get() = %v, want nil or from-fast-worker", got)
	}
	if got := overflowPool.Get(); got != nil && got != "from-overflow-worker" {
		t.Fatalf("overflowPool.Get() = %v, want nil or from-overflow-worker", got)
	}
}

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

func TestPoolHybridFastSlotGrowthAmortized(t *testing.T) {
	// Visiting many pools in order on a fresh thread must reslice within spare
	// capacity instead of reallocating and copying locals.fast on every Put.
	const total = 1024
	pools := make([]sync.Pool, total)
	vals := make([]int, total)
	for i := range vals {
		vals[i] = i
	}

	var before, after runtime.MemStats
	done := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		runtime.ReadMemStats(&before)
		for i := range pools {
			pools[i].Put(&vals[i])
		}
		runtime.ReadMemStats(&after)
		for i := range pools {
			if got := pools[i].Get(); got != &vals[i] {
				t.Errorf("pool[%d].Get() = %v, want %p", i, got, &vals[i])
				break
			}
		}
		close(done)
	}()
	<-done

	if runtime.Compiler == "llgo" && runtime.GOARCH != "wasm" {
		// Geometric doubling up to 1024 entries allocates ~32 KiB for the fast
		// slice plus per-pool local headers (~256 KiB total); quadratic copying
		// allocated ~16.8 MiB.
		if after.TotalAlloc > before.TotalAlloc {
			delta := after.TotalAlloc - before.TotalAlloc
			const maxExpectedBytes = 2 << 20
			if delta > maxExpectedBytes {
				t.Fatalf("first Put across %d pools allocated %d bytes (> %d); fast-slot slice likely reallocated per slot",
					total, delta, maxExpectedBytes)
			}
		}
	}
}
