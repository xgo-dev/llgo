//go:build darwin || (linux && !baremetal) || windows

// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package runtime

import (
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/sync"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

// Lock-free 4-state word for OS address-wait fast paths (Darwin
// os_sync_wait_on_address, Linux FUTEX_WAIT_PRIVATE, Windows WaitOnAddress).
// sigSleeping == 0 matches Darwin's llgo_channel_address_wait expected-zero
// contract and the zero-initial value used by waitOnAddress.
const (
	sigSleeping uint32 = 0 // waiter committed to OS address wait on &notified == 0
	sigPending  uint32 = 1 // waiter spinning or not yet parked
	sigWaking   uint32 = 2 // notifier claimed wake; may be calling OS address wake
	sigDone     uint32 = 3 // notifier done touching chanSignal; safe to reuse/destroy
)

type chanSignal struct {
	mutex     sync.Mutex
	cond      sync.Cond
	notified  uint32
	sleeping  bool
	address   bool
	mutexInit bool
}

var chanMaxSpinners uint32
var chanActiveSpinners uint32

// TryAcquireSpinSlot reserves one slot from the shared core-bounded spin budget
// (exported for runtime subpackages such as platform netpollers).
// atomic.Add returns the previous value, so < maxSpin admits at most maxSpin
// concurrent extended spinners. Callers must pair every true return with
// ReleaseSpinSlot.
func TryAcquireSpinSlot() bool {
	maxSpin := atomic.Load(&chanMaxSpinners)
	if maxSpin == 0 || atomic.Load(&chanActiveSpinners) >= maxSpin {
		return false
	}
	if atomic.Add(&chanActiveSpinners, uint32(1)) < maxSpin {
		return true
	}
	atomic.Sub(&chanActiveSpinners, uint32(1))
	return false
}

// ReleaseSpinSlot releases a slot acquired via TryAcquireSpinSlot.
func ReleaseSpinSlot() {
	atomic.Sub(&chanActiveSpinners, uint32(1))
}

func init() {
	chanAddressWaitInit()
	// Cap extended spinners conservatively (up to 6 on >=12-core hosts) so
	// spinning queue-head waiters never saturate cores or starve workers/GC.
	maxSpin := CPUCount() / 2
	if maxSpin < 1 {
		maxSpin = 1
	}
	if maxSpin > 6 {
		maxSpin = 6
	}
	atomic.Store(&chanMaxSpinners, uint32(maxSpin))
}

func (s *chanSignal) init() {
	s.mutex.Init(nil)
	s.mutexInit = true
	// Retain the backend chosen at initialization. On Darwin, early signals
	// use the condition variable while the address-wait symbols are resolved.
	s.address = chanAddressWaitAvailable()
	if !s.address {
		s.cond.Init(nil)
	}
}

// initFast initializes a waiter signal without creating a mutex when the
// lock-free address-wait path is available.
func (s *chanSignal) initFast() {
	s.address = chanAddressWaitAvailable()
	if s.address {
		atomic.Store(&s.notified, sigPending)
		return
	}
	s.mutex.Init(nil)
	s.mutexInit = true
	s.cond.Init(nil)
}

func (s *chanSignal) useFast() bool {
	return s.address
}

func (s *chanSignal) ensureMutexInit() {
	if !s.mutexInit {
		s.mutex.Init(nil)
		s.mutexInit = true
		if !s.address {
			s.cond.Init(nil)
		}
	}
}

func (s *chanSignal) resetFast() {
	if s.address {
		atomic.Store(&s.notified, sigPending)
	}
}

func (s *chanSignal) lock()   { s.mutex.Lock() }
func (s *chanSignal) unlock() { s.mutex.Unlock() }

// waitFastDone waits until readyFast finishes its OS address-wake call and
// publishes sigDone, guaranteeing the waiter never reuses or destroys
// s.notified while a kernel wake is still referencing it. Because Darwin's
// chanAddressWait expects sigSleeping (0), not sigWaking (2), the rare
// preempted-notifier case falls back to a short c.Usleep(1) after 256 spins.
func (s *chanSignal) waitFastDone() {
	for i := 0; atomic.Load(&s.notified) != sigDone; i++ {
		if i < 256 {
			chanSignalSpinHint()
		} else {
			c.Usleep(1)
		}
	}
}

// parkFast waits for readyFast without any mutex lock/unlock/init/destroy.
//
// spinCooldown packs exponential-backoff state into one lock-free uint32:
//   - 0: hot — extended spinning allowed for queue-head waiters when a spare
//     spinner slot is available (not written while already 0; cleared back to 0
//     on a spin hit after a prior cold backoff).
//   - (step << 16) | rem: cold — skip extended spin while rem > 0, decrementing
//     rem on each sleep; when rem reaches 0, re-probe extended spin and either
//     reset to 0 on hit or double step (2 -> 4 -> 8 -> 16 -> 32 -> 64) on miss.
//
// For single-channel ops, only queue-head waiters (isHead == true) consume
// extended spin slots so waiters queued behind another waiter sleep promptly.
// For Select, the caller passes isHead = true if any selected case is at its
// channel's queue head.
func (s *chanSignal) parkFast(spinCooldown *uint32, isHead bool) {
	if st := atomic.Load(&s.notified); st >= sigWaking {
		if st == sigWaking {
			s.waitFastDone()
		}
		return
	}
	const fastSpin = 32
	firstLimit := chanSignalSpinLimit
	if firstLimit > fastSpin {
		firstLimit = fastSpin
	}
	if !isHead && firstLimit > 16 {
		firstLimit = 16
	}
	for i := 0; i < firstLimit; i++ {
		if st := atomic.Load(&s.notified); st >= sigWaking {
			if spinCooldown != nil && atomic.Load(spinCooldown) != 0 {
				atomic.Store(spinCooldown, uint32(0))
			}
			if st == sigWaking {
				s.waitFastDone()
			}
			return
		}
		chanSignalSpinHint()
	}
	didExtend := false
	var cd uint32
	if spinCooldown != nil {
		cd = atomic.Load(spinCooldown)
	}
	if isHead && chanSignalSpinLimit > firstLimit && (cd&0xffff) == 0 {
		if TryAcquireSpinSlot() {
			didExtend = true
			for i := firstLimit; i < chanSignalSpinLimit; i++ {
				if st := atomic.Load(&s.notified); st >= sigWaking {
					ReleaseSpinSlot()
					if spinCooldown != nil && cd != 0 {
						atomic.Store(spinCooldown, uint32(0))
					}
					if st == sigWaking {
						s.waitFastDone()
					}
					return
				}
				chanSignalSpinHint()
			}
			ReleaseSpinSlot()
		}
	}
	if isHead && spinCooldown != nil {
		if didExtend {
			step := cd >> 16
			if step < 2 {
				step = 2
			} else if step < 64 {
				step <<= 1
			}
			atomic.Store(spinCooldown, (step<<16)|step)
		} else if rem := cd & 0xffff; rem > 0 {
			atomic.CompareAndExchange(spinCooldown, cd, (cd&0xffff0000)|(rem-1))
		}
	}
	TracebackWaiting(true)
	if _, ok := atomic.CompareAndExchange(&s.notified, sigPending, sigSleeping); ok {
		chanAddressSleep(&s.notified)
	}
	s.waitFastDone()
	TracebackWaiting(false)
}

func (s *chanSignal) readyFast() {
	prev := atomic.Exchange(&s.notified, sigWaking)
	if prev == sigSleeping {
		chanAddressNotify(&s.notified)
	}
	atomic.Store(&s.notified, sigDone)
}

func (s *chanSignal) parkIdleFast() {
	if st := atomic.Load(&s.notified); st >= sigWaking {
		if st == sigWaking {
			s.waitFastDone()
		}
		return
	}
	const idleFastSpin = 32
	spinLimit := chanSignalSpinLimit
	if spinLimit > idleFastSpin {
		spinLimit = idleFastSpin
	}
	for i := 0; i < spinLimit; i++ {
		if st := atomic.Load(&s.notified); st >= sigWaking {
			if st == sigWaking {
				s.waitFastDone()
			}
			return
		}
		chanSignalSpinHint()
	}
	if _, ok := atomic.CompareAndExchange(&s.notified, sigPending, sigSleeping); ok {
		chanAddressSleepIdle(&s.notified)
	}
	s.waitFastDone()
}

// chanSlowSpinLimit caps extended spinning on the mutex/cond slow path
// (synctest, pre-14.4 macOS) which lacks parkFast's cooldown and head gating.
const chanSlowSpinLimit = 128

func (s *chanSignal) park() {
	TracebackWaiting(true)
	atomic.Store(&s.notified, uint32(0))
	s.unlock()
	const fastSpin = 32
	slowLimit := chanSignalSpinLimit
	if slowLimit > chanSlowSpinLimit {
		slowLimit = chanSlowSpinLimit
	}
	firstLimit := slowLimit
	if firstLimit > fastSpin {
		firstLimit = fastSpin
	}
	for i := 0; i < firstLimit; i++ {
		if atomic.Load(&s.notified) != 0 && s.mutex.TryLock() {
			TracebackWaiting(false)
			return
		}
		chanSignalSpinHint()
	}
	if slowLimit > firstLimit && TryAcquireSpinSlot() {
		for i := firstLimit; i < slowLimit; i++ {
			if atomic.Load(&s.notified) != 0 && s.mutex.TryLock() {
				ReleaseSpinSlot()
				TracebackWaiting(false)
				return
			}
			chanSignalSpinHint()
		}
		ReleaseSpinSlot()
	}
	s.lock()
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

const chanIdleSpinLimit = 32

// parkIdle parks a pooled worker thread with no attached G, skipping traceback
// state updates while allowing a brief uncontended spin before sleeping outside
// BDWGC's active thread set.
func (s *chanSignal) parkIdle() {
	atomic.Store(&s.notified, uint32(0))
	s.unlock()
	spinLimit := chanSignalSpinLimit
	if spinLimit > chanIdleSpinLimit {
		spinLimit = chanIdleSpinLimit
	}
	for i := 0; i < spinLimit; i++ {
		if atomic.Load(&s.notified) != 0 && s.mutex.TryLock() {
			return
		}
		chanSignalSpinHint()
	}
	s.lock()
	if atomic.Load(&s.notified) == 0 {
		s.sleeping = true
		if s.address {
			s.unlock()
			chanAddressSleepIdle(&s.notified)
			s.lock()
		} else {
			chanCondSleepIdle(s)
		}
		s.sleeping = false
	}
}

func (s *chanSignal) waitOnAddress() {
	s.unlock()
	chanAddressSleep(&s.notified)
	// Briefly spin for s.mutex right after wake so the notifier's unlock can
	// hand off without another kernel sleep, bounded by 32 hints and the
	// global spin-slot cap.
	postWakeSpin := chanSignalSpinLimit
	if postWakeSpin > 32 {
		postWakeSpin = 32
	}
	if postWakeSpin > 0 && TryAcquireSpinSlot() {
		for i := 0; i < postWakeSpin; i++ {
			if s.mutex.TryLock() {
				ReleaseSpinSlot()
				return
			}
			chanSignalSpinHint()
		}
		ReleaseSpinSlot()
	}
	s.lock()
}

func (s *chanSignal) ready() {
	atomic.Store(&s.notified, uint32(1))
	if !s.sleeping {
		return
	}
	if !s.address {
		s.cond.Signal()
		return
	}
	chanAddressNotify(&s.notified)
}

func (s *chanSignal) destroy() {
	if !s.mutexInit {
		return
	}
	if !s.address {
		s.cond.Destroy()
	}
	s.mutex.Destroy()
	s.mutexInit = false
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
