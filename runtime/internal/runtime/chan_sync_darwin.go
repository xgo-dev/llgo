//go:build darwin && !(llgo && baremetal && tinygo.riscv32 && !nogc)

// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package runtime

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
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
	// Retain the backend chosen at initialization. Early signals use the
	// condition variable while the address-wait symbols are being resolved.
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
	// Bound spinning by iterations. After the budget, park on the selected
	// backend if the notifier has not published completion.
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
	// The mutex protects the decision to sleep. The condition wait releases
	// it atomically with parking. The address wait releases it first and
	// then compares the notification word before parking. Both paths return
	// holding the mutex, after the notifier has finished using the signal.
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
			case syscall.EINTR:
				// Recheck the word before retrying the interrupted wait.
			case syscall.ENOMEM, syscall.EFAULT:
				// Back off before retrying under memory pressure or a transient page fault.
				c.Usleep(100)
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
	if !s.sleeping {
		return
	}
	if !s.address {
		s.cond.Signal()
		return
	}
	// There may be no kernel waiter if the notification was published before it parked.
	if chanAddressWake(&s.notified) < 0 && syscall.Errno(os.Errno()) != syscall.ENOENT {
		panic("channel address wake failed")
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
