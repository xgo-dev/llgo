//go:build llgo && js && wasm && llgo.wasm_workers

package runtime

import (
	"github.com/goplus/llgo/runtime/internal/clite/pthread/sync"
	"github.com/goplus/llgo/runtime/internal/clite/sync/atomic"
	"github.com/goplus/llgo/runtime/internal/wasmworkers"
)

type chanMutex struct {
	mutex sync.Mutex
}

func (m *chanMutex) init() {
	if m.mutex.Init(nil) != 0 {
		fatal("runtime: failed to initialize channel mutex")
	}
}

func (m *chanMutex) Lock() {
	m.mutex.Lock()
}

func (m *chanMutex) Unlock() {
	m.mutex.Unlock()
}

type chanSignal struct {
	lockWord uint32
	waiter   SchedulerWaiter
}

func (s *chanSignal) init() {
	s.waiter = CurrentSchedulerWaiter()
}

func (s *chanSignal) lock() {
	for {
		if _, ok := atomic.CompareAndExchange(&s.lockWord, uint32(0), uint32(1)); ok {
			return
		}
		wasmworkers.Wait(&s.lockWord, 1, -1)
	}
}

func (s *chanSignal) unlock() {
	atomic.Store(&s.lockWord, uint32(0))
	wasmworkers.Wake(&s.lockWord)
}

func (s *chanSignal) park() {
	s.unlock()
	s.waiter.Park()
	s.lock()
}

func (s *chanSignal) ready() {
	s.waiter.Ready()
}

func (*chanSignal) destroy() {}

func chanBlockForever() {
	waiter := CurrentSchedulerWaiter()
	waiter.Park()
	fatal("runtime: permanently parked goroutine was resumed")
}
