//go:build llgo && baremetal && tinygo.riscv32 && !nogc

package runtime

// Single-hart cooperative scheduling keeps every channel operation inside one
// running G, so channel locks need no memory synchronization and waiting is a
// scheduler park, mirroring chan_sync_wasm.go.

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

// chanBlockForever parks forever for select on nil channels and similar
// dead ends. NewSynctestWaiter returns nil outside a synctest bubble (the
// nil receiver makes Block a no-op), so on this backend the wait itself is
// the scheduler park in CurrentSchedulerWaiter().Park; a wake would only
// come from goready on this G, which no code path issues, hence the fatal.
func chanBlockForever() {
	NewSynctestWaiter().Block()
	CurrentSchedulerWaiter().Park()
	fatal("runtime: permanently parked goroutine was resumed")
}

// SchedulerWaiter is an opaque handle used by runtime primitives that park
// the current G without exposing scheduler-owned state.
type SchedulerWaiter struct {
	gp *g
}

// CurrentSchedulerWaiter returns a handle for the current G.
func CurrentSchedulerWaiter() SchedulerWaiter {
	return SchedulerWaiter{gp: getg()}
}

// Park suspends the waiter until Ready makes it runnable.
func (w SchedulerWaiter) Park() {
	if w.gp == nil || getg() != w.gp {
		fatal("runtime: invalid baremetal scheduler waiter")
		return
	}
	gopark()
}

// Ready makes a previously parked waiter runnable.
func (w SchedulerWaiter) Ready() {
	if w.gp == nil {
		fatal("runtime: ready of invalid baremetal scheduler waiter")
		return
	}
	goready(w.gp)
}
