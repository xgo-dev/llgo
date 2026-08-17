package syscall

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

//go:linkname c_getenv C.getenv
func c_getenv(name *c.Char) *c.Char

// Note: this follows C.getenv semantics and is not safe to race with
// setenv/putenv calls made from C code on other threads.
func Getenv(key string) (value string, found bool) {
	if len(key) == 0 {
		return "", false
	}
	name := c.AllocaCStr(key)
	v := c_getenv(name)
	if v == nil {
		return "", false
	}
	return c.GoString(v), true
}
