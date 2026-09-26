//go:build !windows

package stacktrace

import _ "unsafe"

const LLGoFiles = "_wrap/traceback_unix.c; _wrap/traceback_accessors.c"

//go:noescape
//go:linkname Bounds C.llgo_traceback_bounds
func Bounds(low, high *uintptr)
