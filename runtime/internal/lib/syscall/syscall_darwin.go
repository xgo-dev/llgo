package syscall

import (
	"syscall"

	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/clite/os"
)

func Syscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno) {
	return RawSyscall(trap, a1, a2, a3)
}

func Syscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err syscall.Errno) {
	return RawSyscall6(trap, a1, a2, a3, a4, a5, a6)
}

func RawSyscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno) {
	ret := c_syscall(c.Long(trap), a1, a2, a3)
	if ret <= -1 {
		return ^uintptr(0), 0, syscall.Errno(os.Errno())
	}
	return uintptr(ret), 0, 0
}

func RawSyscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err syscall.Errno) {
	ret := c_syscall(c.Long(trap), a1, a2, a3, a4, a5, a6)
	if ret <= -1 {
		return ^uintptr(0), 0, syscall.Errno(os.Errno())
	}
	return uintptr(ret), 0, 0
}
