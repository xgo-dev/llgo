package syscall

import (
	_ "unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
)

//go:linkname c_syscall C.syscall
func c_syscall(number c.Long, __llgo_va_list ...any) c.Long
