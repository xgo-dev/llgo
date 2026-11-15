//go:build darwin

// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package runtime

import (
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
}

// Bound the work before falling back to a condition wait. This is a count of
// atomic polls, not a time guarantee, and never repeatedly wakes an idle thread.
const chanSignalSpinLimit = 8192

func (s *chanSignal) init() {
	s.mutex.Init(nil)
	s.cond.Init(nil)
}
func (s *chanSignal) lock()   { s.mutex.Lock() }
func (s *chanSignal) unlock() { s.mutex.Unlock() }

func (s *chanSignal) park() {
	TracebackWaiting(true)
	atomic.Store(&s.notified, uint32(0))
	s.unlock()
	for i := 0; i < chanSignalSpinLimit; i++ {
		if atomic.Load(&s.notified) != 0 {
			break
		}
	}
	s.lock()
	// Reset and the decision to sleep are both protected by mutex. A notifier
	// either publishes during spinning or signals after Wait releases mutex;
	// neither transition can lose a wake. Return holding mutex even if the spin
	// succeeds, so finish completes ready before the waiter can destroy/reuse us.
	if atomic.Load(&s.notified) == 0 {
		s.sleeping = true
		s.cond.Wait(&s.mutex)
		s.sleeping = false
	}
	TracebackWaiting(false)
}

func (s *chanSignal) ready() {
	// The caller holds mutex through the complete notification. Spinning waiters
	// only need the atomic publication; sleeping waiters also need a kernel wake.
	atomic.Store(&s.notified, uint32(1))
	if s.sleeping {
		s.cond.Signal()
	}
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
