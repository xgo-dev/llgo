//go:build !llgo || !wasm || (wasip1 && llgo.wasi_threads)

package runtime

import "github.com/goplus/llgo/runtime/internal/clite/pthread/sync"

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
	s.cond.Wait(&s.mutex)
}

func (s *chanSignal) ready() {
	s.cond.Signal()
}

func (s *chanSignal) destroy() {
	s.cond.Destroy()
	s.mutex.Destroy()
}

func chanBlockForever() {
	var signal chanSignal
	signal.init()
	signal.lock()
	for {
		signal.park()
	}
}
