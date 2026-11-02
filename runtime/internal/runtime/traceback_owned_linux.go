//go:build llgo && linux && !baremetal && !wasm

package runtime

import "unsafe"

//go:linkname markOwned C.llgo_traceback_mark_owned
func markOwned(unsafe.Pointer)

// MarkMainThreadOwned is called only by a standalone executable's entry after
// runtime initialization. A C library constructor runs on an adopted host
// thread; initializing the runtime must not promote that thread to owned.
func MarkMainThreadOwned() { markOwned(getg().tracebackThread) }
