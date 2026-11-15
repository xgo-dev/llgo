//go:build !wasm && !baremetal && !nogc

package gotest

import (
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
)

var gcPercentAllocationSink []byte

//go:noinline
func gcPercentAllocate(count int) {
	for range count {
		gcPercentAllocationSink = make([]byte, 64<<10)
	}
}

func TestGCPercentConcurrentExchange(t *testing.T) {
	old := debug.SetGCPercent(100)
	defer debug.SetGCPercent(old)
	const n = 32
	previous := make(chan int, n)
	var workers sync.WaitGroup
	for i := range n {
		workers.Add(1)
		go func(value int) {
			defer workers.Done()
			previous <- debug.SetGCPercent(value)
		}(101 + i)
	}
	workers.Wait()
	close(previous)
	counts := map[int]int{100: 1}
	for i := range n {
		counts[101+i]++
	}
	for value := range previous {
		counts[value]--
	}
	counts[debug.SetGCPercent(100)]--
	for value, count := range counts {
		if count != 0 {
			t.Fatalf("GC percentage exchanges lost value %d: balance %d", value, count)
		}
	}
}

func TestGCPercentLowersLargeCachedBudget(t *testing.T) {
	old := debug.SetGCPercent(1<<31 - 1)
	defer debug.SetGCPercent(old)
	gcPercentAllocate(16)
	debug.SetGCPercent(100)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	gcPercentAllocate(2048)
	runtime.ReadMemStats(&after)
	if after.NumGC <= before.NumGC {
		t.Fatal("lowering GOGC retained the previous enormous collection threshold")
	}
	gcPercentAllocationSink = nil
}

func TestGCPercentControlsAutomaticCollection(t *testing.T) {
	old := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(old)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	gcPercentAllocate(1024)
	runtime.ReadMemStats(&after)
	if after.NumGC != before.NumGC {
		t.Fatalf("automatic collection continued with GOGC off: %d -> %d", before.NumGC, after.NumGC)
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	if after.NumGC <= before.NumGC {
		t.Fatal("disabling automatic GC also disabled explicit runtime.GC")
	}
	if previous := debug.SetGCPercent(0); previous != -1 {
		t.Fatalf("previous GC percentage = %d, want -1", previous)
	}
	runtime.ReadMemStats(&before)
	gcPercentAllocate(2048)
	runtime.ReadMemStats(&after)
	if after.NumGC <= before.NumGC {
		t.Fatal("automatic collection did not resume")
	}
	gcPercentAllocationSink = nil
}
