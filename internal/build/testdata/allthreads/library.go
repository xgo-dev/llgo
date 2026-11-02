package main

import "syscall"
import _ "unsafe"

//go:linkname raw syscall.runtime_doAllThreadsSyscall
func raw(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, errno uintptr)

//export AllThreadsUnsupported
func AllThreadsUnsupported() int32 {
	r1, r2, err := syscall.AllThreadsSyscall(syscall.SYS_GETPID, 0, 0, 0)
	if err != syscall.ENOTSUP || r1 != ^uintptr(0) || r2 != ^uintptr(0) {
		return 1
	}
	r1, r2, err = syscall.AllThreadsSyscall6(syscall.SYS_SETRESUID, ^uintptr(0), ^uintptr(0), ^uintptr(0), 0, 0, 0)
	if err != syscall.ENOTSUP || r1 != ^uintptr(0) || r2 != ^uintptr(0) {
		return 2
	}
	for _, trap := range []uintptr{syscall.SYS_GETPID, syscall.SYS_SETRESUID} {
		if _, _, errno := raw(trap, ^uintptr(0), ^uintptr(0), ^uintptr(0), 0, 0, 0); errno != uintptr(syscall.ENOTSUP) {
			return 3
		}
	}
	return 0
}

func main() {}
