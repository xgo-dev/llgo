//go:build baremetal

package runtime

import (
	_ "unsafe"
)

const (
	LLGoPackage = "link"
	LLGoFiles   = "_wrap/debugtrap.c"
)

func c_maxprocs() int32 { return 1 }

//go:linkname c_debugtrap C.llgo_debugtrap
func c_debugtrap()
