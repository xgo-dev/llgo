//go:build linux
// +build linux

package gotest

import (
	"testing"

	"golang.org/x/sys/unix"
)

func TestKubeXSysUnixNoErrorSyscalls(t *testing.T) {
	rawPID, _ := unix.RawSyscallNoError(unix.SYS_GETPID, 0, 0, 0)
	if rawPID == 0 {
		t.Fatal("RawSyscallNoError(SYS_GETPID) returned zero pid")
	}

	sysPID, _ := unix.SyscallNoError(unix.SYS_GETPID, 0, 0, 0)
	if sysPID != rawPID {
		t.Fatalf("SyscallNoError pid = %d, want %d", sysPID, rawPID)
	}
}
