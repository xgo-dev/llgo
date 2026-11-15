//go:build linux && !baremetal

package gotest

import (
	"os"
	"syscall"
	"testing"
)

func TestAllThreadsSyscallResults(t *testing.T) {
	for _, six := range []bool{false, true} {
		var r1, r2 uintptr
		var err syscall.Errno
		if six {
			r1, r2, err = syscall.AllThreadsSyscall6(syscall.SYS_GETPID, 0, 0, 0, 0, 0, 0)
		} else {
			r1, r2, err = syscall.AllThreadsSyscall(syscall.SYS_GETPID, 0, 0, 0)
		}
		// Both compilers reject C-linked environments without full thread
		// ownership. Pure Go and LLGo's controlled executables can succeed.
		if err == syscall.ENOTSUP {
			if r1 != ^uintptr(0) || r2 != ^uintptr(0) {
				t.Fatalf("unsupported results = %#x, %#x", r1, r2)
			}
			continue
		}
		if err != 0 || r1 != uintptr(os.Getpid()) || r2 != 0 {
			t.Fatalf("getpid results = %d, %d, %v", r1, r2, err)
		}
	}
}
