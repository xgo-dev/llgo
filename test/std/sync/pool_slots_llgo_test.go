//go:build llgo && !baremetal && (!wasm || llgo.wasi_threads)

package sync_test

import (
	"sync"
	"sync/atomic"
	"testing"
	_ "unsafe"
)

//go:linkname allocatePoolSlot runtime.allocatePoolSlot
func allocatePoolSlot(next *atomic.Uintptr) uintptr

func TestPoolSlotAllocationSaturates(t *testing.T) {
	const limit = 1024
	const available = 2
	const workers = 32
	var next atomic.Uintptr
	next.Store(limit - available)
	var ids [workers]uintptr
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := range ids {
		go func() {
			defer wg.Done()
			<-start
			ids[i] = allocatePoolSlot(&next)
		}()
	}
	close(start)
	wg.Wait()

	var counts [available + 1]int
	for _, id := range ids {
		if id < limit-available || id > limit {
			t.Fatalf("got invalid slot %d near the slot limit", id)
		}
		counts[id-(limit-available)]++
	}
	if counts != [available + 1]int{1, 1, workers - available} {
		t.Fatalf("slot allocation counts = %v, want one owner per remaining fast slot", counts)
	}
	if next.Load() != limit || allocatePoolSlot(&next) != limit || next.Load() != limit {
		t.Fatalf("next slot = %d after overflow allocations, want %d", next.Load(), limit)
	}
}
