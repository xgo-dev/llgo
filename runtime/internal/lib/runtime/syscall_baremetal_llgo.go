//go:build linux && baremetal && llgo.baremetal.stdio && go1.27

package runtime

import _ "unsafe"

//go:linkname syscall_runtime_syscall3 syscall.runtime_syscall3
func syscall_runtime_syscall3(_, _, _, _ uintptr) (uintptr, int32) {
	return ^uintptr(0), 38 // ENOSYS
}

//go:linkname syscall_runtime_syscall6 syscall.runtime_syscall6
func syscall_runtime_syscall6(_, _, _, _, _, _, _ uintptr) (uintptr, int32) {
	return ^uintptr(0), 38 // ENOSYS
}

//go:linkname syscall_rawVforkSyscall syscall.rawVforkSyscall
func syscall_rawVforkSyscall(_, _, _, _ uintptr) (uintptr, uintptr) {
	return ^uintptr(0), 38 // ENOSYS
}

func fcntl(fd int32, cmd int32, arg int32) (int32, int32) {
	if fd < 0 || fd > 2 {
		return -1, 9 // EBADF
	}
	if cmd == 3 { // F_GETFL
		if fd == 0 {
			return 0, 0 // O_RDONLY
		}
		return 1, 0 // O_WRONLY
	}
	return -1, 38 // ENOSYS
}
