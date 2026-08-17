//go:build linux && (amd64 || arm64) && !go1.22
// +build linux
// +build amd64 arm64
// +build !go1.22

package runtime

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	cliteos "github.com/xgo-dev/llgo/runtime/internal/clite/os"
	clitesyscall "github.com/xgo-dev/llgo/runtime/internal/clite/syscall"
)

//go:linkname c_syscall C.syscall
func c_syscall(sysno c.Long, __llgo_va_list ...any) c.Long

//go:linkname syscall_rawVforkSyscall syscall.rawVforkSyscall
func syscall_rawVforkSyscall(trap, a1, a2 uintptr) (r1 uintptr, err uintptr) {
	if trap == uintptr(clitesyscall.SYS_CLONE) {
		pid := cliteos.Fork()
		if pid < 0 {
			return uintptr(int64(pid)), uintptr(cliteos.Errno())
		}
		return uintptr(int64(pid)), 0
	}
	r := c_syscall(c.Long(trap), c.Long(a1), c.Long(a2))
	if r == -1 {
		return uintptr(r), uintptr(cliteos.Errno())
	}
	return uintptr(r), 0
}
