//go:build darwin && !baremetal

package runtime

import (
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	cliteos "github.com/xgo-dev/llgo/runtime/internal/clite/os"
	csyscall "github.com/xgo-dev/llgo/runtime/internal/clite/syscall"
)

func pollPipe(fds *[2]c.Int) bool {
	if cliteos.Pipe(fds) != 0 {
		return false
	}
	if !setCloseOnExec(fds[0]) || !setCloseOnExec(fds[1]) || !setNonblock(fds[0]) || !setNonblock(fds[1]) {
		cliteos.Close(fds[0])
		cliteos.Close(fds[1])
		return false
	}
	return true
}

func setCloseOnExec(fd c.Int) bool {
	flags := cliteos.Fcntl(fd, c.Int(csyscall.F_GETFD))
	return flags >= 0 && cliteos.Fcntl(fd, c.Int(csyscall.F_SETFD), uintptr(flags)|uintptr(csyscall.FD_CLOEXEC)) >= 0
}

func setNonblock(fd c.Int) bool {
	flags := cliteos.Fcntl(fd, c.Int(csyscall.F_GETFL))
	return flags >= 0 && cliteos.Fcntl(fd, c.Int(csyscall.F_SETFL), uintptr(flags)|uintptr(csyscall.O_NONBLOCK)) >= 0
}
