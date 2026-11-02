//go:build !baremetal && !wasm && !windows

package runtime

import (
	_ "unsafe"
)

const (
	LLGoPackage = "link"
	LLGoFiles   = "_wrap/runtime.c; _wrap/debugtrap.c; _wrap/fault.c; _wrap/dynunwind.c; _wrap/profile.c; _wrap/signal.c; _wrap/timer_wait.c; _wrap/libcall.c"
)

//go:linkname c_maxprocs C.llgo_maxprocs
func c_maxprocs() int32

//go:linkname c_debugtrap C.llgo_debugtrap
func c_debugtrap()
