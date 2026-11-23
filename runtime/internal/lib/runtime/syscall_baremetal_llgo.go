//go:build baremetal && !llgo.baremetal.stdio

package runtime

import _ "unsafe"

// Baremetal images have no kernel syscall surface: the masqueraded GOOS
// still selects the syscall package's runtime hooks, so provide ENOSYS
// stubs instead of routing through the C syscall bridge.
//
//go:linkname syscall_runtime_syscall3 syscall.runtime_syscall3
func syscall_runtime_syscall3(trap, a1, a2, a3 uintptr) (r1 uintptr, errno int32) {
	_, _, _ = trap, a1, a2
	_ = a3
	return 0, 38 // ENOSYS
}

//go:linkname syscall_runtime_syscall6 syscall.runtime_syscall6
func syscall_runtime_syscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1 uintptr, errno int32) {
	_, _, _ = trap, a1, a2
	_, _, _ = a3, a4, a5
	_ = a6
	return 0, 38 // ENOSYS
}

func fcntl(fd int32, cmd int32, arg int32) (int32, int32) {
	_, _, _ = fd, cmd, arg
	return -1, 38 // ENOSYS
}

//go:linkname syscall_rawVforkSyscall syscall.rawVforkSyscall
func syscall_rawVforkSyscall(trap, a1, a2, a3 uintptr) (r1 uintptr, err uintptr) {
	_, _, _, _ = trap, a1, a2, a3
	return 0, 38 // ENOSYS
}
