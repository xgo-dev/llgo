//go:build llgo && wasm && !(wasip1 && llgo.wasi_threads)

package runtime

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
		fatal("runtime: invalid WebAssembly scheduler waiter")
		return
	}
	gopark()
}

// Ready makes a previously parked waiter runnable.
func (w SchedulerWaiter) Ready() {
	if w.gp == nil {
		fatal("runtime: ready of invalid WebAssembly scheduler waiter")
		return
	}
	goready(w.gp)
}
