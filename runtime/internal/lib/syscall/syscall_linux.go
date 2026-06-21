package syscall

import (
	c "github.com/goplus/llgo/runtime/internal/clite"
)

func rawSyscallNoError(trap, a1, a2, a3 uintptr) (r1, r2 uintptr) {
	ret := c_syscall(c.Long(trap), a1, a2, a3)
	if ret <= -1 {
		return ^uintptr(0), 0
	}
	return uintptr(ret), 0
}
