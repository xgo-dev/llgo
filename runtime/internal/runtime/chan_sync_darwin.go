//go:build darwin

// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package runtime

import (
	_ "unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/clite/os"
	"github.com/xgo-dev/llgo/runtime/internal/clite/syscall"
	"github.com/xgo-dev/llgo/runtime/internal/sync"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

type chanMutex struct{ mutex sync.UnfairMutex }

// Channel queues provide their own waiter ordering. A fair-share pthread
// mutex forces an OS handoff between contenders even for these short critical
// sections; os_unfair_lock can acquire the channel without that handoff.
func (m *chanMutex) init()   {}
func (m *chanMutex) Lock()   { m.mutex.Lock() }
func (m *chanMutex) Unlock() { m.mutex.Unlock() }

type chanSignal struct {
	mutex    sync.Mutex
	cond     sync.Cond
	notified uint32
	sleeping bool
	address  bool
}

const chanSignalLLGoFiles = "; _wrap/chan_sync_darwin.c"

//go:linkname bindChanAddressWait C.llgo_bind_channel_address_wait
func bindChanAddressWait() int32

//go:linkname chanAddressWait C.llgo_channel_address_wait
func chanAddressWait(notified *uint32) int32

//go:linkname chanAddressWake C.llgo_channel_address_wake
func chanAddressWake(notified *uint32) int32

//go:linkname chanSignalSpinHint llgo.asm
func chanSignalSpinHint(instruction string)

var chanAddressWaitEnabled uint32

func init() {
	if bindChanAddressWait() != 0 {
		atomic.Store(&chanAddressWaitEnabled, 1)
	}
}

func (s *chanSignal) init() {
	s.mutex.Init(nil)
	// Keep the selection for the signal's lifetime, including signals
	// initialized during early runtime startup.
	s.address = atomic.Load(&chanAddressWaitEnabled) != 0
	if !s.address {
		s.cond.Init(nil)
	}
}
func (s *chanSignal) lock()   { s.mutex.Lock() }
func (s *chanSignal) unlock() { s.mutex.Unlock() }

func (s *chanSignal) park() {
	TracebackWaiting(true)
	atomic.Store(&s.notified, uint32(0))
	s.unlock()
	// The architecture bounds iterations with a CPU relax hint, not elapsed
	// time. After the budget, fall back to the ordinary condition wait.
	for i := 0; i < chanSignalSpinLimit; i++ {
		if atomic.Load(&s.notified) != 0 && s.mutex.TryLock() {
			// ready publishes before the notifier unlocks. Keep trying until
			// that unlock instead of immediately joining a fair-share wait.
			TracebackWaiting(false)
			return
		}
		chanSignalSpinHint(chanSignalSpinInstruction)
	}
	s.lock()
	// Reset and the decision to sleep are both protected by mutex. A notifier
	// either publishes during spinning or signals after Wait releases mutex;
	// neither transition can lose a wake. Return holding mutex even if the spin
	// succeeds, so finish completes ready before the waiter can destroy/reuse us.
	if atomic.Load(&s.notified) == 0 {
		s.sleeping = true
		if s.address {
			s.waitOnAddress()
		} else {
			s.cond.Wait(&s.mutex)
		}
		s.sleeping = false
	}
	TracebackWaiting(false)
}

func (s *chanSignal) waitOnAddress() {
	// The kernel compares the notification word before sleeping. A wake
	// between unlock and that comparison leaves the word visibly set.
	s.unlock()
	for atomic.Load(&s.notified) == 0 {
		if chanAddressWait(&s.notified) < 0 {
			switch syscall.Errno(os.Errno()) {
			case syscall.EINTR, syscall.ENOMEM, syscall.EFAULT:
				// Recheck the notification word before a transient retry.
			default:
				panic("channel address wait failed")
			}
		}
	}
	// The notifier must release the lock before this signal can be reused.
	s.lock()
}

func (s *chanSignal) ready() {
	// The caller holds mutex through the complete notification. Spinning waiters
	// only need the atomic publication; sleeping waiters also need a kernel wake.
	atomic.Store(&s.notified, uint32(1))
	if s.sleeping {
		if !s.address {
			s.cond.Signal()
		} else if chanAddressWake(&s.notified) < 0 && syscall.Errno(os.Errno()) != syscall.ENOENT {
			// A notifier may publish before the waiter enters the kernel.
			panic("channel address wake failed")
		}
	}
}

func (s *chanSignal) destroy() {
	if !s.address {
		s.cond.Destroy()
	}
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
