//go:build darwin

// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package runtime

import (
	"unsafe"

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

// Lock-free 4-state word for Darwin's os_sync_wait_on_address fast path.
// sigSleeping == 0 matches llgo_channel_address_wait's expected-zero contract.
const (
	sigSleeping uint32 = 0 // waiter committed to os_sync_wait_on_address(&notified, 0)
	sigPending  uint32 = 1 // waiter spinning or not yet parked
	sigWaking   uint32 = 2 // notifier claimed wake; may be calling wake_by_address
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
var chanMaxSpinners uint32
var chanActiveSpinners uint32

// TryAcquireSpinSlot reserves one slot from the shared core-bounded spin budget.
// Callers must pair every true return with ReleaseSpinSlot.
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
	if bindChanAddressWait() != 0 {
		atomic.Store(&chanAddressWaitEnabled, 1)
	}
	// Cap extended spinners conservatively (up to 6 on >=12-core hosts) so
	// spinning queue-head waiters never saturate P-cores or starve workers/GC.
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
	// Retain the backend chosen at initialization. Early signals use the
	// condition variable while the address-wait symbols are being resolved.
	s.address = atomic.Load(&chanAddressWaitEnabled) != 0
	if !s.address {
		s.cond.Init(nil)
	}
}

// initFast initializes a waiter signal without creating a pthread_mutex when
// Darwin's os_sync_wait_on_address lock-free path is available.
func (s *chanSignal) initFast() {
	s.address = atomic.Load(&chanAddressWaitEnabled) != 0
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

func (s *chanSignal) waitFastDone() {
	for i := 0; atomic.Load(&s.notified) != sigDone; i++ {
		if i < 256 {
			chanSignalSpinHint(chanSignalSpinInstruction)
		} else {
			c.Usleep(1)
		}
	}
}

func (s *chanSignal) sleepAddressFast() {
	for atomic.Load(&s.notified) == sigSleeping {
		if chanAddressWait(&s.notified) < 0 {
			switch syscall.Errno(os.Errno()) {
			case syscall.EINTR:
			case syscall.ENOMEM, syscall.EFAULT:
				c.Usleep(100)
			default:
				panic("channel address wait failed")
			}
		}
	}
}

// parkFast waits for readyFast without any pthread mutex lock/unlock/init/destroy.
//
// spinCooldown packs exponential-backoff state into one uint32:
//   - 0: hot — extended spinning allowed for queue-head waiters when a spare
//     spinner slot is available (never written on hot hits -> zero cache-line bounce).
//   - (step << 16) | rem: cold — skip extended spin while rem > 0, decrementing
//     rem on each sleep; when rem reaches 0, re-probe extended spin and either
//     reset to 0 on hit or double step (2 -> 4 -> 8 -> 16 -> 32 -> 64) on miss.
//
// Only queue-head waiters (isHead == true) consume extended spin slots; waiters
// queued behind another waiter on the same FIFO channel sleep promptly so
// spinners never starve active reply/handoff waiters.
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
		chanSignalSpinHint(chanSignalSpinInstruction)
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
				chanSignalSpinHint(chanSignalSpinInstruction)
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
		s.sleepAddressFast()
	}
	s.waitFastDone()
	TracebackWaiting(false)
}

func (s *chanSignal) readyFast() {
	prev := atomic.Exchange(&s.notified, sigWaking)
	if prev == sigSleeping {
		if chanAddressWake(&s.notified) < 0 && syscall.Errno(os.Errno()) != syscall.ENOENT {
			panic("channel address wake failed")
		}
	}
	atomic.Store(&s.notified, sigDone)
}

//llgo:cdecl
func parkIdleBlockingDarwinFast(arg unsafe.Pointer) unsafe.Pointer {
	s := (*chanSignal)(arg)
	s.sleepAddressFast()
	return nil
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
		chanSignalSpinHint(chanSignalSpinInstruction)
	}
	if _, ok := atomic.CompareAndExchange(&s.notified, sigPending, sigSleeping); ok {
		mPoolDoBlocking(parkIdleBlockingDarwinFast, unsafe.Pointer(s))
	}
	s.waitFastDone()
}

func (s *chanSignal) park() {
	TracebackWaiting(true)
	atomic.Store(&s.notified, uint32(0))
	s.unlock()
	const fastSpin = 32
	firstLimit := chanSignalSpinLimit
	if firstLimit > fastSpin {
		firstLimit = fastSpin
	}
	for i := 0; i < firstLimit; i++ {
		if atomic.Load(&s.notified) != 0 && s.mutex.TryLock() {
			TracebackWaiting(false)
			return
		}
		chanSignalSpinHint(chanSignalSpinInstruction)
	}
	if chanSignalSpinLimit > firstLimit {
		if maxSpin := atomic.Load(&chanMaxSpinners); maxSpin > 0 && atomic.Load(&chanActiveSpinners) < maxSpin {
			if atomic.Add(&chanActiveSpinners, uint32(1)) < maxSpin {
				for i := firstLimit; i < chanSignalSpinLimit; i++ {
					if atomic.Load(&s.notified) != 0 && s.mutex.TryLock() {
						atomic.Sub(&chanActiveSpinners, uint32(1))
						TracebackWaiting(false)
						return
					}
					chanSignalSpinHint(chanSignalSpinInstruction)
				}
			}
			atomic.Sub(&chanActiveSpinners, uint32(1))
		}
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

//llgo:cdecl
func parkIdleBlockingDarwin(arg unsafe.Pointer) unsafe.Pointer {
	s := (*chanSignal)(arg)
	if s.address {
		s.waitOnAddress()
	} else {
		s.cond.Wait(&s.mutex)
	}
	return nil
}

// parkIdle parks a pooled worker thread with no attached G, skipping traceback
// state updates while allowing a brief uncontended spin before sleeping outside
// BDWGC's active thread set via GC_do_blocking.
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
		chanSignalSpinHint(chanSignalSpinInstruction)
	}
	s.lock()
	if atomic.Load(&s.notified) == 0 {
		s.sleeping = true
		mPoolDoBlocking(parkIdleBlockingDarwin, unsafe.Pointer(s))
		s.sleeping = false
	}
}

func (s *chanSignal) waitOnAddress() {
	s.unlock()
	for atomic.Load(&s.notified) == 0 {
		if chanAddressWait(&s.notified) < 0 {
			switch syscall.Errno(os.Errno()) {
			case syscall.EINTR:
			case syscall.ENOMEM, syscall.EFAULT:
				c.Usleep(100)
			default:
				panic("channel address wait failed")
			}
		}
	}
	postWakeSpin := chanSignalSpinLimit
	if postWakeSpin > 32 {
		postWakeSpin = 32
	}
	if maxSpin := atomic.Load(&chanMaxSpinners); maxSpin > 0 && atomic.Load(&chanActiveSpinners) < maxSpin {
		if atomic.Add(&chanActiveSpinners, uint32(1)) < maxSpin {
			for i := 0; i < postWakeSpin; i++ {
				if s.mutex.TryLock() {
					atomic.Sub(&chanActiveSpinners, uint32(1))
					return
				}
				chanSignalSpinHint(chanSignalSpinInstruction)
			}
		}
		atomic.Sub(&chanActiveSpinners, uint32(1))
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
	if chanAddressWake(&s.notified) < 0 && syscall.Errno(os.Errno()) != syscall.ENOENT {
		panic("channel address wake failed")
	}
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
