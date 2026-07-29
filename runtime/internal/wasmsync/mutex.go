//go:build llgo && js && wasm && llgo.wasm_workers

// Package wasmsync provides synchronization primitives that can cooperate
// with the WebAssembly worker scheduler while contended.
package wasmsync

import (
	"github.com/goplus/llgo/runtime/internal/clite/sync/atomic"
	"github.com/goplus/llgo/runtime/internal/wasmworkers"
)

const mutexWaitNanoseconds = int64(1_000_000)

// Mutex is a zero-value-ready lock for worker-shared runtime state.
type Mutex struct {
	state uint32
}

// Lock acquires m. A contended lock periodically calls yield so a worker
// blocked behind an allocator can acknowledge a stop-the-world request.
func (m *Mutex) Lock(yield func()) {
	for {
		if _, ok := atomic.CompareAndExchange(&m.state, uint32(0), uint32(1)); ok {
			return
		}
		if yield != nil {
			yield()
		}
		wasmworkers.Wait(&m.state, 1, mutexWaitNanoseconds)
	}
}

// Unlock releases m and wakes all waiters.
func (m *Mutex) Unlock() {
	atomic.Store(&m.state, uint32(0))
	wasmworkers.Wake(&m.state)
}
