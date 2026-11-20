//go:build linux && baremetal && llgo.baremetal.stdio

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	cliteos "github.com/xgo-dev/llgo/runtime/internal/clite/os"
)

//go:linkname syscall_runtime_syscall3 syscall.runtime_syscall3
func syscall_runtime_syscall3(trap, a1, a2, a3 uintptr) (uintptr, int32) {
	r, _, errno := llgoBaremetalSyscall6(trap, a1, a2, a3, 0, 0, 0)
	return r, int32(errno)
}

//go:linkname syscall_runtime_syscall6 syscall.runtime_syscall6
func syscall_runtime_syscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (uintptr, int32) {
	r, _, errno := llgoBaremetalSyscall6(trap, a1, a2, a3, a4, a5, a6)
	return r, int32(errno)
}

// The bare-metal stdlib uses the Linux/ARM syscall ABI. Both layouts of
// internal/runtime/syscall and the public syscall wrappers share this console.
func llgoBaremetalSyscall6(trap, fd, buf, size, _, _, _ uintptr) (uintptr, uintptr, uintptr) {
	const (
		armRead  = 3
		armWrite = 4
		badFD    = 9
		ioError  = 5
		noSys    = 38
	)
	var n c.SsizeT
	switch trap {
	case armRead:
		if fd != 0 {
			return ^uintptr(0), 0, badFD
		}
		n = cliteos.Read(c.Int(fd), unsafe.Pointer(buf), size)
	case armWrite:
		if fd != 1 && fd != 2 {
			return ^uintptr(0), 0, badFD
		}
		n = cliteos.Write(c.Int(fd), unsafe.Pointer(buf), size)
	default:
		return ^uintptr(0), 0, noSys
	}
	if n < 0 {
		return ^uintptr(0), 0, ioError
	}
	return uintptr(n), 0, 0
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
