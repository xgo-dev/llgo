//go:build darwin

// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package sync

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const LLGoFiles = "_wrap/sync_darwin.c"

// UnfairMutex has the native layout of os_unfair_lock and starts unlocked.
// Its owner must unlock on the same native thread; condition-variable waits
// continue to use Mutex. Darwin's hosted goroutines each own a native thread.
type UnfairMutex struct{ state uint32 }

//go:linkname unfairMutexLock C.os_unfair_lock_lock
func unfairMutexLock(m *UnfairMutex)

//go:linkname unfairMutexTryLock C.os_unfair_lock_trylock
func unfairMutexTryLock(m *UnfairMutex) bool

//go:linkname unfairMutexUnlock C.os_unfair_lock_unlock
func unfairMutexUnlock(m *UnfairMutex)

func (m *UnfairMutex) Lock()         { unfairMutexLock(m) }
func (m *UnfairMutex) TryLock() bool { return unfairMutexTryLock(m) }
func (m *UnfairMutex) Unlock()       { unfairMutexUnlock(m) }

//go:linkname darwinHasWaitUint32 C.llgo_darwin_has_wait_uint32
func darwinHasWaitUint32() c.Int

//go:linkname darwinHasWaitUint32Timeout C.llgo_darwin_has_wait_uint32_timeout
func darwinHasWaitUint32Timeout() c.Int

//go:linkname darwinWaitUint32 C.llgo_darwin_wait_uint32
func darwinWaitUint32(addr *uint32, value uint32) c.Int

//go:linkname darwinWaitUint32Timeout C.llgo_darwin_wait_uint32_timeout
func darwinWaitUint32Timeout(addr *uint32, value uint32, timeoutNs uint64) c.Int

//go:linkname darwinWakeUint32 C.llgo_darwin_wake_uint32
func darwinWakeUint32(addr *uint32)

//go:linkname darwinWakeUint32All C.llgo_darwin_wake_uint32_all
func darwinWakeUint32All(addr *uint32)

//go:linkname darwinWaitErrnoRetryable C.llgo_darwin_wait_errno_retryable
func darwinWaitErrnoRetryable(err c.Int) c.Int

// HasWaitUint32 reports whether Darwin's os_sync_wait_on_address API (macOS 14.4+) is available.
func HasWaitUint32() bool { return darwinHasWaitUint32() != 0 }

// HasWaitUint32Timeout reports whether Darwin's timed os_sync_wait_on_address API is available.
func HasWaitUint32Timeout() bool { return darwinHasWaitUint32Timeout() != 0 }

// WaitUint32 atomically checks addr and sleeps while it contains value.
// Returns an errno value (0 on success, ENOPROTOOPT if unavailable).
// Callers must always re-read *addr after WaitUint32 returns (spurious wakes
// and EINTR are possible).
func WaitUint32(addr *uint32, value uint32) c.Int {
	return darwinWaitUint32(addr, value)
}

// WaitUint32Timeout sleeps while *addr == value for up to timeoutNs nanoseconds.
// Returns an errno value (0 on success, ETIMEDOUT on timeout or timeoutNs == 0
// without inspecting *addr, ENOPROTOOPT if unavailable). Callers must always
// re-read *addr after WaitUint32Timeout returns.
func WaitUint32Timeout(addr *uint32, value uint32, timeoutNs uint64) c.Int {
	return darwinWaitUint32Timeout(addr, value, timeoutNs)
}

// WakeUint32 wakes one thread waiting for addr.
func WakeUint32(addr *uint32) {
	darwinWakeUint32(addr)
}

// WakeUint32All wakes all threads waiting for addr.
func WakeUint32All(addr *uint32) {
	darwinWakeUint32All(addr)
}

// WaitErrnoRetryable reports whether err is a benign WaitUint32 /
// WaitUint32Timeout return (0, EINTR, EAGAIN, or ETIMEDOUT) that callers may
// retry immediately without backoff.
func WaitErrnoRetryable(err c.Int) bool {
	return darwinWaitErrnoRetryable(err) != 0
}
