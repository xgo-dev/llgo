package main

import (
	"syscall"
	_ "unsafe"
)

const LLGoFiles = "_wrap/probe.c"

//go:linkname raw syscall.runtime_doAllThreadsSyscall
func raw(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, errno uintptr)

//go:linkname pause C.llgo_test_allthreads_pause
func pause()

//export LLGoHostSyscallStatus
func LLGoHostSyscallStatus() int32 {
	if _, _, err := syscall.AllThreadsSyscall(syscall.SYS_GETPID, 0, 0, 0); err != syscall.ENOTSUP {
		return -1
	}
	if _, _, err := syscall.AllThreadsSyscall6(syscall.SYS_SETRESUID, ^uintptr(0), ^uintptr(0), ^uintptr(0), 0, 0, 0); err != syscall.ENOTSUP {
		return -2
	}
	// A harmless no-op accepted by both main and the fix. This must succeed
	// for an owned executable main and reject an adopted C-library host.
	_, _, errno := raw(syscall.SYS_SETRESUID, ^uintptr(0), ^uintptr(0), ^uintptr(0), 0, 0, 0)
	return int32(errno)
}

func main() {
	for i := 0; i < 50; i++ {
		err := LLGoHostSyscallStatus()
		if err == 0 {
			println("allthreads-owned-main-ok")
			return
		}
		if err != int32(syscall.EAGAIN) && err != int32(syscall.ENOTSUP) {
			panic("unexpected all-thread syscall error")
		}
		pause()
	}
	panic("standalone main was not registered as owned")
}
