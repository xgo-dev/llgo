//go:build windows

// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/sync"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

type chanMutex struct {
	mutex sync.Mutex
}

// Windows sync.Mutex is backed by a zero-initialized SRWLOCK.
func (m *chanMutex) init()   { m.mutex.Init(nil) }
func (m *chanMutex) Lock()   { m.mutex.Lock() }
func (m *chanMutex) Unlock() { m.mutex.Unlock() }

func chanAddressWaitInit() {}

func chanAddressWaitAvailable() bool {
	// Win32 GDB cannot unwind suspended Go workers through the native
	// WaitOnAddress stack. Keep the supported condition-variable path there.
	return unsafe.Sizeof(uintptr(0)) == 8
}

func chanAddressSleep(notified *uint32) {
	for atomic.Load(notified) == sigSleeping {
		if err := sync.WaitUint32(notified, sigSleeping); err != 0 && atomic.Load(notified) == sigSleeping {
			c.Usleep(100)
		}
	}
}

// chanAddressSleepIdle uses WaitUint32Inactive so idle worker threads park
// outside BDWGC's active set on every Windows ABI (including MSVC ARM64)
// without nesting GC_do_blocking.
func chanAddressSleepIdle(notified *uint32) {
	for atomic.Load(notified) == sigSleeping {
		if err := sync.WaitUint32Inactive(notified, sigSleeping); err != 0 && atomic.Load(notified) == sigSleeping {
			c.Usleep(100)
		}
	}
}

func chanAddressNotify(notified *uint32) {
	sync.WakeUint32(notified)
}

func chanCondSleepIdle(s *chanSignal) {
	s.cond.WaitInactive(&s.mutex)
}
