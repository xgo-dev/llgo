//go:build llgo && js && wasm && llgo.wasm.workers

package runtime

import (
	"github.com/xgo-dev/llgo/runtime/internal/wasmsync"
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

func (s *chanSignal) initFast()              { s.init() }
func (s *chanSignal) useFast() bool          { return false }
func (s *chanSignal) ensureMutexInit()       {}
func (s *chanSignal) resetFast()             {}
func (s *chanSignal) parkFast(*uint32, bool) {}
func (s *chanSignal) readyFast()             {}
func (s *chanSignal) parkIdleFast()          {}

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
	NewSynctestWaiter().Block()
	waiter := CurrentSchedulerWaiter()
	waiter.Park()
	fatal("runtime: permanently parked goroutine was resumed")
}
