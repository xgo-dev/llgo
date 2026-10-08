//go:build llgo && js && wasm && llgo.wasm.workers

// Package wasmsync provides synchronization primitives that cooperate with
// the WebAssembly worker scheduler while contended.
package wasmsync

import (
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
	"github.com/xgo-dev/llgo/runtime/internal/wasmworkers"
)

// A bounded wait lets a contending worker revisit yield for a GC stop even if
// the lock holder cannot unlock until that worker acknowledges the stop.
const mutexWaitNanoseconds = int64(1_000_000)

// Mutex is a zero-value-ready lock for worker-shared runtime state.
type Mutex struct {
	state   uint32
	waiters uint32
}

// Lock acquires m. A contended lock periodically calls yield so a worker
// blocked behind the allocator can acknowledge a stop-the-world request. The
// bounded wait is necessary because a GC request does not unlock this mutex.
func (m *Mutex) Lock(yield func()) {
	if _, ok := atomic.CompareAndExchange(&m.state, uint32(0), uint32(1)); ok {
		return
	}
	atomic.Add(&m.waiters, uint32(1))
	for {
		if _, ok := atomic.CompareAndExchange(&m.state, uint32(0), uint32(1)); ok {
			atomic.Sub(&m.waiters, uint32(1))
			return
		}
		if yield != nil {
			yield()
		}
		wasmworkers.Wait(&m.state, 1, mutexWaitNanoseconds)
	}
}

// Unlock releases m and wakes one waiter only if contention was observed.
func (m *Mutex) Unlock() {
	atomic.Store(&m.state, uint32(0))
	if atomic.Load(&m.waiters) != 0 {
		wasmworkers.WakeOne(&m.state)
	}
}

// Contended reports whether another worker is waiting for this mutex.
func (m *Mutex) Contended() bool { return atomic.Load(&m.waiters) != 0 }
