//go:build llgo && linux && !baremetal && !wasm

package runtime

import "unsafe"

//go:linkname markOwned C.llgo_traceback_mark_owned
func markOwned(unsafe.Pointer)

func markTracebackOwned(gp *g) { markOwned(gp.tracebackThread) }
