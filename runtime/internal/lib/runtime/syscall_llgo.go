//go:build darwin || linux

package runtime

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	cliteos "github.com/xgo-dev/llgo/runtime/internal/clite/os"
)

//go:linkname syscall_runtime_syscall3 syscall.runtime_syscall3
func syscall_runtime_syscall3(trap, a1, a2, a3 uintptr) (r1 uintptr, errno int32) {
	ret := c_syscallN(c.Long(trap), a1, a2, a3)
	return syscallResult(ret)
}

//go:linkname syscall_runtime_syscall6 syscall.runtime_syscall6
func syscall_runtime_syscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1 uintptr, errno int32) {
	ret := c_syscallN(c.Long(trap), a1, a2, a3, a4, a5, a6)
	return syscallResult(ret)
}

func syscallResult(ret c.Long) (r1 uintptr, errno int32) {
	if ret <= -1 {
		return ^uintptr(0), int32(cliteos.Errno())
	}
	return uintptr(ret), 0
}

//go:linkname c_syscallN C.syscall
func c_syscallN(number c.Long, __llgo_va_list ...any) c.Long
