//go:build llgo && !baremetal && !wasm

package sync_test

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"testing"
	"time"
)

// Each hybrid test starts with an independent process-wide slot counter.
func runPoolHybridInFreshProcess(t *testing.T) bool {
	t.Helper()
	const key = "LLGO_POOL_HYBRID_TEST"
	if os.Getenv(key) == t.Name() {
		return true
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "-test.run=^"+t.Name()+"$")
	cmd.Env = append(os.Environ(), key+"="+t.Name())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("hybrid pool subprocess: %v\n%s", err, out)
	}
	return false
}

func TestPoolHybridFastAndOverflowSlots(t *testing.T) {
	if !runPoolHybridInFreshProcess(t) {
		return
	}
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
}

func TestPoolHybridFastSlotGrowthAmortized(t *testing.T) {
	if !runPoolHybridInFreshProcess(t) {
		return
	}
	// Visiting every fast slot in order must allocate each page once.
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

	if delta := after.TotalAlloc - before.TotalAlloc; delta > 2<<20 {
		t.Fatalf("first Put across %d fast slots allocated %d bytes", total, delta)
	}
}

func TestPoolHybridSparseLateSlot(t *testing.T) {
	if !runPoolHybridInFreshProcess(t) {
		return
	}
	// Reserve the early slots on the test thread, then have fresh threads
	// touch only a distant slot.
	early := make([]sync.Pool, 900)
	for i := range early {
		early[i].Put(&early[i])
	}
	var late sync.Pool
	late.Put(&late)
	const workers = 32
	ready := make(chan struct{}, workers)
	start := make(chan struct{})
	done := make(chan struct{}, workers)
	for range workers {
		go func() {
			runtime.LockOSThread()
			ready <- struct{}{}
			<-start
			late.Put(&late)
			if got := late.Get(); got != &late {
				t.Errorf("late pool returned %v, want %p", got, &late)
			}
			done <- struct{}{}
		}()
	}
	for range workers {
		<-ready
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	close(start)
	for range workers {
		<-done
	}
	runtime.ReadMemStats(&after)
	if delta := after.TotalAlloc - before.TotalAlloc; delta > 128<<10 {
		t.Fatalf("touching one late slot on %d threads allocated %d bytes", workers, delta)
	}
}
