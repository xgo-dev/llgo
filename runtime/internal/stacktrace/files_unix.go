//go:build !windows

package stacktrace

import _ "unsafe"

const LLGoFiles = "_wrap/traceback_unix.c; _wrap/traceback_accessors.c"

//go:noescape
//go:linkname Bounds C.llgo_traceback_bounds
func Bounds(low, high *uintptr)

//go:linkname allThreadsControlled C.llgo_traceback_allthreads_controlled
func allThreadsControlled() int32

// AllThreadsControlled reports whether Linux controlled all-threads syscall
// mode is active and requires every live OS thread to stay registered.
func AllThreadsControlled() bool { return allThreadsControlled() != 0 }
