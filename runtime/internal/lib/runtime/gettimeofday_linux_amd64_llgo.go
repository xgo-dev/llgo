//go:build linux && amd64 && !baremetal

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	cliteos "github.com/xgo-dev/llgo/runtime/internal/clite/os"
	clitesyscall "github.com/xgo-dev/llgo/runtime/internal/clite/syscall"
)

//go:linkname c_gettimeofday C.gettimeofday
func c_gettimeofday(tv *clitesyscall.Timeval, tz unsafe.Pointer) c.Int

// syscall.Gettimeofday and syscall.Time call an amd64 assembly helper. LLGo's
// replacement syscall package does not include that symbol, so forward it to
// libc while preserving syscall's errno result.
//
//go:linkname syscall_gettimeofday syscall.gettimeofday
func syscall_gettimeofday(tv *clitesyscall.Timeval) clitesyscall.Errno {
	if c_gettimeofday(tv, nil) != 0 {
		return clitesyscall.Errno(cliteos.Errno())
	}
	return 0
}
