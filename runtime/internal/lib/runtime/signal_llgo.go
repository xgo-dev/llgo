//go:build !baremetal && !wasm && !windows

package runtime

import (
	latomic "sync/atomic"
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
)

// Unix signal handlers atomically mark signals pending and write wake tokens to
// a nonblocking self-pipe. A dedicated runtime goroutine claims the pending
// signals and enters Go code in a normal execution context; the
// async-signal-safe C handler never calls into Go.

type sigState struct {
	active  bool
	ignored bool
}

const (
	signalCount = 65 // max across Unix systems; matches os/signal
	signalWords = (signalCount + 31) / 32
)

const (
	// Raw ABI shared with LLGO_SIGNAL_* in _wrap/signal.c and
	// LLGO_PROF_SIGNAL_* in _wrap/profile.c. Keep the values synchronized.
	signalModeDefault int32 = iota
	signalModeWanted
	signalModeIgnored
)

var (
	sigInitState uint32
	sigReadFD    c.Int

	sigPipeInheritedIgnored bool

	sigMu     psync.Mutex
	sigCond   psync.Cond
	sigWaitMu psync.Mutex
	sigQueue  [signalWords]uint32
	sigRecv   [signalWords]uint32
	sigStates map[uint32]*sigState

	// sigReceiving is true only while the os/signal receive goroutine is
	// blocked here waiting for another signal. Reaching that state after a
	// pipe barrier proves that it has processed every earlier queue entry.
	sigReceiving  bool
	sigBarrierAck uint64
)

//go:linkname c_signalInit C.llgo_signal_init
func c_signalInit() c.Int

//go:linkname c_signalEnable C.llgo_signal_enable
func c_signalEnable(sig c.Int) c.Int

//go:linkname c_signalDisable C.llgo_signal_disable
func c_signalDisable(sig c.Int) c.Int

//go:linkname c_signalIgnore C.llgo_signal_ignore
func c_signalIgnore(sig c.Int) c.Int

//go:linkname c_signalIgnorePipe C.llgo_signal_ignore_pipe
func c_signalIgnorePipe() c.Int

//go:linkname c_signalRaisePipe C.llgo_signal_raise_pipe
func c_signalRaisePipe() c.Int

//go:linkname c_signalDiePipe C.llgo_signal_die_pipe
func c_signalDiePipe() c.Int

//go:linkname c_signalBarrier C.llgo_signal_barrier
func c_signalBarrier() c.Int

//go:linkname c_signalRecv C.llgo_signal_recv
func c_signalRecv(fd c.Int, sig *c.Int) c.Int

func init() {
	// Go programs turn SIGPIPE from ordinary file descriptors into EPIPE.
	// Install that process-wide baseline before user goroutines can write.
	previous := c_signalIgnorePipe()
	if previous < 0 {
		panic("runtime: failed to ignore SIGPIPE")
	}
	sigPipeInheritedIgnored = previous != 0
}

// signalPipe implements the explicit SIGPIPE raised by package os after an
// EPIPE write to stdout or stderr. By contrast, EPIPE on ordinary descriptors
// never reaches here; those writes rely on the ignored baseline from init.
func signalPipe() {
	const sigPipe = 13 // SIGPIPE on the supported Unix hosts.

	// OS signal delivery initializes the queue before it installs handlers.
	// A stdout write can use the inherited SIGPIPE policy before that queue
	// exists, including in programs that never use os/signal.
	if latomic.LoadUint32(&sigInitState) != sigInitDone {
		if !sigPipeInheritedIgnored {
			dieWithSIGPIPE()
		}
		return
	}
	sigMu.Lock()
	defer sigMu.Unlock()
	st := sigStates[sigPipe]
	switch {
	case st == nil && sigPipeInheritedIgnored:
		return
	case st != nil && st.ignored:
		return
	case st != nil && st.active:
		if c_signalRaisePipe() != 0 {
			panic("runtime: failed to raise SIGPIPE")
		}
		return
	default:
		dieWithSIGPIPE()
	}
}

func dieWithSIGPIPE() {
	if c_signalDiePipe() != 0 {
		panic("runtime: failed to terminate with SIGPIPE")
	}
}

const (
	sigInitUninit uint32 = iota
	sigInitDone
	sigInitBusy
)

func ensureSignalInit() {
	for {
		switch state := latomic.LoadUint32(&sigInitState); state {
		case sigInitDone:
			return
		case sigInitUninit:
			if latomic.CompareAndSwapUint32(&sigInitState, sigInitUninit, sigInitBusy) {
				sigMu.Init(nil)
				sigCond.Init(nil)
				sigWaitMu.Init(nil)
				sigStates = make(map[uint32]*sigState)
				sigReadFD = c_signalInit()
				if sigReadFD < 0 {
					panic("runtime: failed to initialize signal pipe")
				}
				go signalReadLoop()
				latomic.StoreUint32(&sigInitState, sigInitDone)
				return
			}
		}
		c.Usleep(1)
	}
}

func signalReadLoop() {
	for {
		var value c.Int
		if c_signalRecv(sigReadFD, &value) != 0 {
			panic("runtime: failed to read signal pipe")
		}
		if value == 0 {
			sigMu.Lock()
			sigBarrierAck++
			sigCond.Broadcast()
			sigMu.Unlock()
			continue
		}
		signalCallback(uint32(value))
	}
}

func signalCallback(sig uint32) {
	if sig == 0 || sig >= signalCount {
		return
	}
	sigMu.Lock()
	// Every positive value was claimed from a pending bit set by a handler that
	// was active at the instant the signal arrived. Preserve it even if Reset or
	// Stop has since changed sigStates; os/signal keeps stopping channels
	// registered until signalWaitUntilIdle observes the barrier below.
	sigQueue[sig/32] |= 1 << (sig & 31)
	sigCond.Broadcast()
	sigMu.Unlock()
}

func signalBitsEmpty(bits *[signalWords]uint32) bool {
	for _, word := range bits {
		if word != 0 {
			return false
		}
	}
	return true
}

func ensureSignalState(sig uint32) *sigState {
	st := sigStates[sig]
	if st == nil {
		st = &sigState{}
		sigStates[sig] = st
	}
	return st
}

func setSignalHandler(sig uint32, enabled bool) c.Int {
	locked := cpuProfileSignalLock(sig)
	var code c.Int
	mode := signalModeDefault
	if enabled {
		mode = signalModeWanted
	}
	if handled, updateCode := cpuProfileSignalUpdate(sig, mode); handled {
		code = c.Int(updateCode)
	} else if enabled {
		code = c_signalEnable(c.Int(sig))
	} else {
		code = c_signalDisable(c.Int(sig))
	}
	cpuProfileSignalUnlock(locked)
	return code
}

func setSignalIgnored(sig uint32) c.Int {
	locked := cpuProfileSignalLock(sig)
	var code c.Int
	if handled, updateCode := cpuProfileSignalUpdate(sig, signalModeIgnored); handled {
		code = c.Int(updateCode)
	} else {
		code = c_signalIgnore(c.Int(sig))
	}
	cpuProfileSignalUnlock(locked)
	return code
}

// signal_enable enables Go signal delivery for sig.
func signal_enable(sig uint32) {
	ensureSignalInit()
	sigMu.Lock()
	st := ensureSignalState(sig)
	if (!st.active || st.ignored) && setSignalHandler(sig, true) != 0 {
		sigMu.Unlock()
		panic("runtime: failed to install signal handler")
	}
	st.active = true
	st.ignored = false
	sigMu.Unlock()
}

// signal_disable disables Go signal delivery for sig.
func signal_disable(sig uint32) {
	ensureSignalInit()
	sigMu.Lock()
	st := sigStates[sig]
	if st == nil || !st.active {
		sigMu.Unlock()
		return
	}
	st.active = false
	st.ignored = false
	if setSignalHandler(sig, false) != 0 {
		sigMu.Unlock()
		panic("runtime: failed to restore signal handler")
	}
	sigMu.Unlock()
}

// signal_ignore causes sig to be ignored (do not deliver to Go).
func signal_ignore(sig uint32) {
	ensureSignalInit()
	sigMu.Lock()
	st := ensureSignalState(sig)
	if (!st.active || !st.ignored) && setSignalIgnored(sig) != 0 {
		sigMu.Unlock()
		panic("runtime: failed to install ignored signal handler")
	}
	st.active = true
	st.ignored = true
	sigMu.Unlock()
}

// signal_ignored reports whether sig is being ignored.
func signal_ignored(sig uint32) bool {
	ensureSignalInit()
	sigMu.Lock()
	st := sigStates[sig]
	ignored := st != nil && st.ignored
	sigMu.Unlock()
	return ignored
}

// signal_recv receives the next queued signal.
func signal_recv() uint32 {
	ensureSignalInit()
	sigMu.Lock()
	for {
		for sig := uint32(1); sig < signalCount; sig++ {
			bit := uint32(1) << (sig & 31)
			if sigRecv[sig/32]&bit != 0 {
				sigRecv[sig/32] &^= bit
				sigMu.Unlock()
				return sig
			}
		}

		for signalBitsEmpty(&sigQueue) {
			sigReceiving = true
			sigCond.Broadcast()
			sigCond.Wait(&sigMu)
		}
		sigReceiving = false
		sigRecv = sigQueue
		sigQueue = [signalWords]uint32{}
	}
}

func signalWaitUntilIdle() {
	ensureSignalInit()

	// os/signal documents this as single-caller runtime machinery, but keep
	// the LLGo implementation serialized so concurrent Stop calls cannot
	// mistake another caller's pipe marker for their own.
	sigWaitMu.Lock()
	defer sigWaitMu.Unlock()

	sigMu.Lock()
	wantAck := sigBarrierAck + 1
	sigMu.Unlock()
	if c_signalBarrier() != 0 {
		panic("runtime: failed to synchronize signal pipe")
	}

	sigMu.Lock()
	for sigBarrierAck < wantAck || !signalBitsEmpty(&sigQueue) ||
		!signalBitsEmpty(&sigRecv) || !sigReceiving {
		sigCond.Wait(&sigMu)
	}
	sigMu.Unlock()
}
