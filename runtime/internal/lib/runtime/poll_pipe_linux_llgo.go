//go:build linux && !baremetal

package runtime

import (
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	csyscall "github.com/xgo-dev/llgo/runtime/internal/clite/syscall"
	_ "unsafe"
)

//go:linkname c_pipe2 C.pipe2
func c_pipe2(fds *[2]c.Int, flags c.Int) c.Int

func pollPipe(fds *[2]c.Int) bool {
	// Set flags atomically, avoiding both repeated fcntl calls and inheritance
	// of a wake descriptor by a concurrent fork/exec.
	return c_pipe2(fds, c.Int(csyscall.O_CLOEXEC)|c.Int(csyscall.O_NONBLOCK)) == 0
}
