//go:build llgo && !baremetal && !nogc

package gotest

import (
	"runtime"
	"testing"
	_ "unsafe"
)

//go:linkname getBDWGCFinalizeOnDemand C.GC_get_finalize_on_demand
func getBDWGCFinalizeOnDemand() int32

//go:linkname setBDWGCFinalizeOnDemand C.GC_set_finalize_on_demand
func setBDWGCFinalizeOnDemand(enabled int32)

func TestRuntimeGCDrainsBDWGCFinalizersOnDemand(t *testing.T) {
	// BDWGC normally may invoke ready finalizers during a later allocation.
	// On-demand mode makes runtime.GC's explicit drain observable without
	// relying on allocation timing. This setting is process-global, so this
	// test and the neighboring finalizer tests must remain sequential.
	old := getBDWGCFinalizeOnDemand()
	setBDWGCFinalizeOnDemand(1)
	t.Cleanup(func() {
		setBDWGCFinalizeOnDemand(old)
	})

	const n = 32
	finalized := make(chan int32, n)
	created := make(chan struct{})
	go func() {
		makeFinalizerTinyObjects(n, finalized)
		close(created)
	}()
	<-created

	// InvokeFinalizers runs synchronously once BDWGC has queued an object.
	// The retries only let the producer goroutine exit so that conservative
	// stack and register roots no longer keep the objects reachable.
	for range 8 {
		runtime.Gosched()
		runGCWithTimeout(t)
		if len(finalized) > n/2 {
			return
		}
	}
	if got := len(finalized); got <= n/2 {
		t.Fatalf("runtime.GC ran only %d/%d on-demand finalizers", got, n)
	}
}
