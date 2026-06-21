//go:build !baremetal

package runtime

import (
	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/clite/libuv"
	psync "github.com/goplus/llgo/runtime/internal/clite/pthread/sync"
	latomic "github.com/goplus/llgo/runtime/internal/lib/sync/atomic"
)

// Minimal signal support for stdlib os/signal on llgo/darwin.

type sigState struct {
	handle  libuv.Signal
	inited  bool
	active  bool
	ignored bool
}

var (
	sigInitState uint32

	sigMu     psync.Mutex
	sigCond   psync.Cond
	sigQueue  []uint32
	sigStates map[uint32]*sigState
)

func ensureSignalInit() {
	const (
		sigInitUninit uint32 = iota
		sigInitDone
		sigInitBusy
	)
	for {
		state := latomic.LoadUint32(&sigInitState)
		switch state {
		case sigInitDone:
			return
		case sigInitUninit:
			if latomic.CompareAndSwapUint32(&sigInitState, sigInitUninit, sigInitBusy) {
				ensureTimerLoop()
				sigMu.Init(nil)
				sigCond.Init(nil)
				sigStates = make(map[uint32]*sigState)
				latomic.StoreUint32(&sigInitState, sigInitDone)
				return
			}
		}
		c.Usleep(1)
	}
}

//export llgo_runtime_signalCallback
func llgo_runtime_signalCallback(handle *libuv.Signal, signum c.Int) {
	signalCallback(handle, signum)
}

func signalCallback(handle *libuv.Signal, signum c.Int) {
	sig := uint32(signum)
	sigMu.Lock()
	st := sigStates[sig]
	if st == nil || !st.active || st.ignored {
		sigMu.Unlock()
		return
	}
	sigQueue = append(sigQueue, sig)
	sigCond.Signal()
	sigMu.Unlock()
}

func ensureSignalState(sig uint32) *sigState {
	st := sigStates[sig]
	if st == nil {
		st = &sigState{}
		sigStates[sig] = st
	}
	return st
}

func startSignalWatcher(sig uint32, st *sigState) {
	if !st.inited {
		submitTimerWork(func() bool {
			checkUV("uv_signal_init", int(libuv.SignalInit(timerLoop, &st.handle)))
			return true
		})
		st.inited = true
	}
	submitTimerWork(func() bool {
		checkUV("uv_signal_start", int(libuv.SignalStartRuntime(&st.handle, c.Int(sig))))
		return true
	})
}

// signal_enable enables Go signal delivery for sig.
func signal_enable(sig uint32) {
	ensureSignalInit()
	var st *sigState
	var doStart bool
	sigMu.Lock()
	st = ensureSignalState(sig)
	st.ignored = false
	if !st.active {
		st.active = true
		doStart = true
	}
	sigMu.Unlock()
	if doStart {
		startSignalWatcher(sig, st)
	}
}

// signal_disable disables Go signal delivery for sig.
func signal_disable(sig uint32) {
	ensureSignalInit()
	var st *sigState
	var doStop bool
	sigMu.Lock()
	st = sigStates[sig]
	if st == nil {
		sigMu.Unlock()
		return
	}
	if st.active && !st.ignored {
		st.active = false
		doStop = true
	}
	sigMu.Unlock()
	if doStop {
		submitTimerWork(func() bool {
			checkUV("uv_signal_stop", int(st.handle.Stop()))
			return true
		})
	}
}

// signal_ignore causes sig to be ignored (do not deliver to Go).
func signal_ignore(sig uint32) {
	ensureSignalInit()
	var st *sigState
	var doStart bool
	sigMu.Lock()
	st = ensureSignalState(sig)
	st.ignored = true
	if !st.active {
		st.active = true
		doStart = true
	}
	sigMu.Unlock()
	if doStart {
		startSignalWatcher(sig, st)
	}
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
	for len(sigQueue) == 0 {
		sigCond.Wait(&sigMu)
	}
	sig := sigQueue[0]
	sigQueue = sigQueue[1:]
	sigMu.Unlock()
	return sig
}

// signalWaitUntilIdle waits until queued signals are drained.
func signalWaitUntilIdle() {
	ensureSignalInit()
	for {
		sigMu.Lock()
		empty := len(sigQueue) == 0
		sigMu.Unlock()
		if empty {
			return
		}
		c.Usleep(1)
	}
}
