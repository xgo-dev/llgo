//go:build llgo && js && wasm && llgo.wasm_workers

package runtime

import (
	"github.com/goplus/llgo/runtime/internal/wasmsync"
)

type chanMutex struct {
	mutex wasmsync.Mutex
}

func (*chanMutex) init() {}

func (m *chanMutex) Lock() {
	m.mutex.Lock(CooperativeSafepoint)
}

func (m *chanMutex) Unlock() {
	m.mutex.Unlock()
}

type chanSignal struct {
	mutex  wasmsync.Mutex
	waiter SchedulerWaiter
}

func (s *chanSignal) init() {
	s.waiter = CurrentSchedulerWaiter()
}

func (s *chanSignal) lock() {
	s.mutex.Lock(CooperativeSafepoint)
}

func (s *chanSignal) unlock() {
	s.mutex.Unlock()
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
