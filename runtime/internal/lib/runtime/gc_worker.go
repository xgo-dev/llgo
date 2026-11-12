//go:build windows && 386 && !nogc && !baremetal && !wasm

package runtime

import (
	"sync/atomic"

	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
)

// A stable collecting thread avoids repeated collector handoffs between the
// many short-lived native threads used by Windows/386 goroutines. Concurrent
// or reentrant requests retain the synchronous path to avoid callback cycles.
var explicitGC struct {
	state     uint32 // 0: uninitialized, 1: starting, 2: ready
	mu        psync.Mutex
	work      psync.Cond
	done      psync.Cond
	busy      bool
	requested uint64
	completed uint64
}

func requestExplicitGC() {
	if atomic.LoadUint32(&explicitGC.state) != 2 {
		if !atomic.CompareAndSwapUint32(&explicitGC.state, 0, 1) {
			// Starting the worker can allocate and invoke a foreign C
			// finalizer. Reentry must not wait for that same initialization.
			collectExplicitGCBatch()
			return
		}
		explicitGC.mu.Init(nil)
		explicitGC.work.Init(nil)
		explicitGC.done.Init(nil)
		go runExplicitGC()
		atomic.StoreUint32(&explicitGC.state, 2)
	}
	explicitGC.mu.Lock()
	if explicitGC.busy {
		explicitGC.mu.Unlock()
		// BDWGC may run arbitrary C finalizers inside GC_gcollect or
		// GC_invoke_finalizers. They can wait for another goroutine that
		// calls GC. Preserve the existing synchronous path during a batch,
		// rather than queueing that goroutine behind the callback it wakes.
		collectExplicitGCBatch()
		return
	}
	// Publish busy before starting the batch, covering its complete callback
	// lifetime. Each admitted request receives two cycles starting afterward.
	explicitGC.busy = true
	explicitGC.requested++
	request := explicitGC.requested
	explicitGC.work.Signal()
	for explicitGC.completed < request {
		explicitGC.done.Wait(&explicitGC.mu)
	}
	explicitGC.mu.Unlock()
}

func runExplicitGC() {
	markSystemGoroutine()
	for {
		explicitGC.mu.Lock()
		for explicitGC.completed == explicitGC.requested {
			explicitGC.work.Wait(&explicitGC.mu)
		}
		request := explicitGC.requested
		explicitGC.mu.Unlock()
		collectExplicitGCBatch()
		explicitGC.mu.Lock()
		explicitGC.completed = request
		explicitGC.busy = false
		explicitGC.done.Broadcast()
		explicitGC.mu.Unlock()
	}
}
