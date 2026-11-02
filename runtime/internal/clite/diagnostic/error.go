package diagnostic

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// Error writes a host error code without the general C formatter.
// It uses only stack memory and C stdio, including before runtime startup.
func Error(prefix string, code c.Int) {
	buf := c.Alloca(uintptr(len(prefix) + 32))
	i := uintptr(len(prefix) + 32)
	i--
	*(*byte)(unsafe.Add(buf, i)) = '\n'
	i--
	*(*byte)(unsafe.Add(buf, i)) = ')'
	u := uint64(int64(code))
	negative := code < 0
	if negative {
		u = 0 - u
	}
	for {
		i--
		*(*byte)(unsafe.Add(buf, i)) = '0' + byte(u%10)
		u /= 10
		if u == 0 {
			break
		}
	}
	if negative {
		i--
		*(*byte)(unsafe.Add(buf, i)) = '-'
	}
	i -= 7
	c.Memcpy(unsafe.Add(buf, i), unsafe.Pointer(c.Str("(error=")), 7)
	if len(prefix) != 0 {
		i -= uintptr(len(prefix))
		c.Memcpy(unsafe.Add(buf, i), unsafe.Pointer(unsafe.StringData(prefix)), uintptr(len(prefix)))
	}
	c.Fwrite(unsafe.Add(buf, i), 1, uintptr(len(prefix)+32)-i, c.Stderr)
}
