//go:build baremetal || wasm

package runtime

import (
	_ "unsafe"
)

const (
	LLGoPackage = "link"
	LLGoFiles   = "_wrap/debugtrap.c" + wasmFinalizerLLGoFiles + wasiThreadsTimerLLGoFiles
)

//go:linkname c_debugtrap C.llgo_debugtrap
func c_debugtrap()
