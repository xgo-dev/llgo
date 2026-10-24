//go:build linux && (amd64 || arm64)

package syscall_test

import (
	"syscall"
	"testing"
	"time"
	"unsafe"
)

func TestGettimeofdayAndRawFallback(t *testing.T) {
	before := time.Now().Unix()
	// Exercise libc's error path first. A successful time call must not
	// accidentally report the errno left by that preceding syscall.
	r1, _, errno := syscall.RawSyscall(^uintptr(0), 0, 0, 0)
	if r1 != ^uintptr(0) || errno != syscall.ENOSYS {
		t.Fatalf("invalid syscall = %#x, %v; want -1, ENOSYS", r1, errno)
	}
	var libc, raw syscall.Timeval
	if err := syscall.Gettimeofday(&libc); err != nil {
		t.Fatal(err)
	}
	// A direct kernel call bypasses libc's vDSO dispatch and supplies the
	// reference fallback without adding a production vDSO override.
	r1, _, errno = syscall.RawSyscall(syscall.SYS_GETTIMEOFDAY, uintptr(unsafe.Pointer(&raw)), 0, 0)
	if r1 != 0 || errno != 0 {
		t.Fatalf("raw gettimeofday = %d, %v", r1, errno)
	}
	after := time.Now().Unix()
	for _, tv := range []syscall.Timeval{libc, raw} {
		if int64(tv.Sec) < before || int64(tv.Sec) > after || tv.Usec < 0 || tv.Usec >= 1000000 {
			t.Fatalf("gettimeofday = %+v, time interval [%d, %d]", tv, before, after)
		}
	}
}
