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

const chanSignalLLGoFiles = "; _wrap/chan_sync_darwin.c"

//go:linkname bindChanAddressWait C.llgo_bind_channel_address_wait
func bindChanAddressWait() int32

//go:linkname chanAddressWait C.llgo_channel_address_wait
func chanAddressWait(notified *uint32) int32

//go:linkname chanAddressWake C.llgo_channel_address_wake
func chanAddressWake(notified *uint32) int32

var chanAddressWaitEnabled uint32

func chanAddressWaitInit() {
	if bindChanAddressWait() != 0 {
		atomic.Store(&chanAddressWaitEnabled, 1)
	}
}

func chanAddressWaitAvailable() bool {
	return atomic.Load(&chanAddressWaitEnabled) != 0
}

func chanAddressSleep(notified *uint32) {
	for atomic.Load(notified) == sigSleeping {
		if chanAddressWait(notified) < 0 {
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

//llgo:cdecl
func parkIdleBlockingDarwinAddress(arg unsafe.Pointer) unsafe.Pointer {
	chanAddressSleep((*uint32)(arg))
	return nil
}

func chanAddressSleepIdle(notified *uint32) {
	mPoolDoBlocking(parkIdleBlockingDarwinAddress, unsafe.Pointer(notified))
}

func chanAddressNotify(notified *uint32) {
	// ENOENT means the waiter has not entered (or already left) the kernel wait
	// on notified; publishing the new state before calling wake ensures it
	// observes the notification safely.
	if chanAddressWake(notified) < 0 && syscall.Errno(os.Errno()) != syscall.ENOENT {
		panic("channel address wake failed")
	}
}

//llgo:cdecl
func parkIdleBlockingDarwinCond(arg unsafe.Pointer) unsafe.Pointer {
	s := (*chanSignal)(arg)
	s.cond.Wait(&s.mutex)
	return nil
}

func chanCondSleepIdle(s *chanSignal) {
	mPoolDoBlocking(parkIdleBlockingDarwinCond, unsafe.Pointer(s))
}
