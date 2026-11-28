//go:build linux && !baremetal

// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/syscall"
	"github.com/xgo-dev/llgo/runtime/internal/sync"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

type chanMutex struct {
	mutex sync.Mutex
}

func (m *chanMutex) init()   { m.mutex.Init(nil) }
func (m *chanMutex) Lock()   { m.mutex.Lock() }
func (m *chanMutex) Unlock() { m.mutex.Unlock() }

func chanAddressWaitInit() {}

func chanAddressWaitAvailable() bool { return true }

//llgo:cdecl
func chanLinuxSleepBlocking(arg unsafe.Pointer) unsafe.Pointer {
	notified := (*uint32)(arg)
	for atomic.Load(notified) == sigSleeping {
		err := sync.WaitUint32(notified, sigSleeping)
		// EAGAIN leaves *notified != sigSleeping (rechecked below); EINTR
		// retries immediately; back off only on persistent kernel errors.
		if err != 0 && err != c.Int(syscall.EINTR) && atomic.Load(notified) == sigSleeping {
			c.Usleep(100)
		}
	}
	return nil
}

// chanAddressSleep marks the Linux blocking futex wait inactive for BDWGC
// (matching sync_cond_gc.go's bdwgc.CondWait) so STW collections neither
// interrupt blocked channel waiters with suspend signals nor scan wait frames.
func chanAddressSleep(notified *uint32) {
	mPoolDoBlocking(chanLinuxSleepBlocking, unsafe.Pointer(notified))
}

func chanAddressSleepIdle(notified *uint32) {
	mPoolDoBlocking(chanLinuxSleepBlocking, unsafe.Pointer(notified))
}

func chanAddressNotify(notified *uint32) {
	sync.WakeUint32(notified)
}

func chanCondSleepIdle(s *chanSignal) {
	s.cond.WaitInactive(&s.mutex)
}
