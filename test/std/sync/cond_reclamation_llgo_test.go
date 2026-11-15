//go:build llgo && wasip1 && llgo.wasi_threads && !nogc

package sync_test

import (
	"runtime"
	"sync"
	"testing"
)

func TestCondStateReclamation(t *testing.T) {
	// Warm up lazy runtime services and the state-table storage before measuring.
	exerciseCondStates()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < 4; i++ {
		exerciseCondStates()
		runtime.GC()
	}
	runtime.ReadMemStats(&after)
	if after.HeapObjects > before.HeapObjects+16 {
		t.Fatalf("idle Cond states retained heap objects: %d -> %d", before.HeapObjects, after.HeapObjects)
	}
}

//go:noinline
func exerciseCondStates() {
	// Keep all addresses distinct within a batch, including when the allocator
	// reuses the previous batch. Neither operation has a waiter to preserve.
	conds := make([]sync.Cond, 64)
	for i := range conds {
		conds[i].Signal()
		conds[i].Broadcast()
	}
	runtime.KeepAlive(conds)
}
