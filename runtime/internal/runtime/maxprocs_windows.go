//go:build windows

package runtime

import _ "unsafe"

const maxprocsLLGoFiles = "; _wrap/maxprocs_windows.c"

//go:linkname c_maxprocs C.llgo_maxprocs
func c_maxprocs() int32
