//go:build !baremetal

package runtime

import (
	_ "unsafe"
)

const (
	LLGoPackage = "link"
	LLGoFiles   = "_wrap/runtime.c; _wrap/debugtrap.c"
)

//go:linkname c_maxprocs C.llgo_maxprocs
func c_maxprocs() int32

//go:linkname c_debugtrap C.llgo_debugtrap
func c_debugtrap()
