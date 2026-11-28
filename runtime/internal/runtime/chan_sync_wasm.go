//go:build llgo && wasm && !(wasip1 && llgo.wasi_threads) && !llgo.wasm.workers

package runtime

type chanMutex struct{}

func (*chanMutex) init()   {}
func (*chanMutex) Lock()   {}
func (*chanMutex) Unlock() {}

type chanSignal struct {
	waiter SchedulerWaiter
}

func (s *chanSignal) init() {
	s.waiter = CurrentSchedulerWaiter()
}

func (s *chanSignal) initFast()              { s.init() }
func (s *chanSignal) useFast() bool          { return false }
func (s *chanSignal) ensureMutexInit()       { s.waiter = CurrentSchedulerWaiter() }
func (s *chanSignal) resetFast()             {}
func (s *chanSignal) parkFast(*uint32, bool) {}
func (s *chanSignal) readyFast()             {}
func (s *chanSignal) parkIdleFast()          {}

func (*chanSignal) lock()   {}
func (*chanSignal) unlock() {}

func (s *chanSignal) park() {
	s.waiter.Park()
}

func (s *chanSignal) ready() {
	s.waiter.Ready()
}

func (*chanSignal) destroy() {}

func chanBlockForever() {
	NewSynctestWaiter().Block()
	CurrentSchedulerWaiter().Park()
	fatal("runtime: permanently parked goroutine was resumed")
}
