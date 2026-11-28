//go:build linux && !baremetal

package sync

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const LLGoFiles = "_wrap/sync_linux.c"

//go:linkname linuxWaitUint32 C.llgo_linux_wait_uint32
func linuxWaitUint32(addr *uint32, value uint32) c.Int

//go:linkname linuxWaitUint32Timeout C.llgo_linux_wait_uint32_timeout
func linuxWaitUint32Timeout(addr *uint32, value uint32, timeoutNs uint64) c.Int

//go:linkname linuxWakeUint32 C.llgo_linux_wake_uint32
func linuxWakeUint32(addr *uint32)

//go:linkname linuxWaitErrnoRetryable C.llgo_linux_wait_errno_retryable
func linuxWaitErrnoRetryable(err c.Int) c.Int

// WaitUint32 atomically checks addr and sleeps while it contains value.
// The caller rechecks after wakeups, signals, or a mismatched value.
// The return value is zero on success or the syscall errno.
func WaitUint32(addr *uint32, value uint32) c.Int {
	return linuxWaitUint32(addr, value)
}

// WaitUint32Timeout sleeps while *addr == value for up to timeoutNs nanoseconds
// (measured against CLOCK_MONOTONIC via FUTEX_WAIT_PRIVATE). Callers must
// always re-read *addr after WaitUint32Timeout returns.
// The return value is zero on success or the syscall errno.
func WaitUint32Timeout(addr *uint32, value uint32, timeoutNs uint64) c.Int {
	return linuxWaitUint32Timeout(addr, value, timeoutNs)
}

// WakeUint32 wakes one thread waiting for addr.
func WakeUint32(addr *uint32) {
	linuxWakeUint32(addr)
}

// WaitErrnoRetryable reports whether err is a benign WaitUint32 /
// WaitUint32Timeout return (0, EINTR, EAGAIN, or ETIMEDOUT) that callers may
// retry immediately without backoff.
func WaitErrnoRetryable(err c.Int) bool {
	return linuxWaitErrnoRetryable(err) != 0
}
