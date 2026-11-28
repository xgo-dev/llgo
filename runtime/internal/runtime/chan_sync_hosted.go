//go:build !darwin && (!llgo || !wasm || (wasip1 && llgo.wasi_threads))

package runtime

import (
	"github.com/xgo-dev/llgo/runtime/internal/sync"
)

type chanMutex struct {
	mutex sync.Mutex
}

func (m *chanMutex) init() {
	m.mutex.Init(nil)
}

func (m *chanMutex) Lock() {
	m.mutex.Lock()
}

func (m *chanMutex) Unlock() {
	m.mutex.Unlock()
}

type chanSignal struct {
	mutex sync.Mutex
	cond  sync.Cond
}

func (s *chanSignal) init() {
	s.mutex.Init(nil)
	s.cond.Init(nil)
}

func (s *chanSignal) lock() {
	s.mutex.Lock()
}

func (s *chanSignal) unlock() {
	s.mutex.Unlock()
}

func (s *chanSignal) park() {
	TracebackWaiting(true)
	s.cond.Wait(&s.mutex)
	TracebackWaiting(false)
}

// parkIdle waits for the next pooled goroutine assignment without touching
// traceback wait state. WaitInactive always wraps the blocking syscall in a
// single BDWGC GC_do_blocking frame (including on Windows ARM64 MSVC, where
// Wait stays active for LLDB Go-frame unwinding) without nesting GC_do_blocking.
func (s *chanSignal) parkIdle() {
	s.cond.WaitInactive(&s.mutex)
}

func (s *chanSignal) ready() {
	s.cond.Signal()
}

func (s *chanSignal) destroy() {
	s.cond.Destroy()
	s.mutex.Destroy()
}

func chanBlockForever() {
	NewSynctestWaiter().Block()
	var signal chanSignal
	signal.init()
	signal.lock()
	for {
		signal.park()
	}
}
