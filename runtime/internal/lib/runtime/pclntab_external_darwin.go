//go:build llgo_pclntab_external && darwin

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const externalPCLNGOOS = uint8(2)

//go:linkname externalNSGetExecutablePath C._NSGetExecutablePath
func externalNSGetExecutablePath(buf *c.Char, size *uint32) c.Int

func externalPCLNExecutablePath() string {
	size := uint32(1024)
	for size <= 1<<20 {
		buf := make([]byte, int(size))
		capacity := uint32(len(buf))
		if externalNSGetExecutablePath((*c.Char)(unsafe.Pointer(&buf[0])), &capacity) == 0 {
			n := 0
			for n < len(buf) && buf[n] != 0 {
				n++
			}
			return string(buf[:n])
		}
		if capacity <= size {
			return ""
		}
		size = capacity
	}
	return ""
}
