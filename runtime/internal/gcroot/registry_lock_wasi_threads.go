//go:build llgo && wasip1 && wasm && llgo.wasm.gc.linear && llgo.wasi_threads

package gcroot

import _ "unsafe"

const LLGoFiles = "_wrap/registry_wasi_threads.c"

//go:linkname lockRegistry C.llgo_gcroot_lock
func lockRegistry()

//go:linkname unlockRegistry C.llgo_gcroot_unlock
func unlockRegistry()
