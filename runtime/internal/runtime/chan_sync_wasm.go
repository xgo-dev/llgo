//go:build llgo && wasm && !(wasip1 && llgo.wasi_threads)

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
	CurrentSchedulerWaiter().Park()
	fatal("runtime: permanently parked goroutine was resumed")
}
