//go:build llgo_pclntab_external && linux

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	cliteos "github.com/xgo-dev/llgo/runtime/internal/clite/os"
)

const externalPCLNGOOS = uint8(1)

func externalPCLNExecutablePath() string {
	path := c.Str("/proc/self/exe")
	for size := 256; size <= 1<<20; size *= 2 {
		buf := make([]byte, size)
		n := cliteos.Readlink(path, unsafe.Pointer(&buf[0]), uintptr(len(buf)))
		if n < 0 {
			return ""
		}
		if n < len(buf) {
			return string(buf[:n])
		}
	}
	return ""
}
