//go:build !windows && !darwin && !wasm && !baremetal

package bdwgc

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const blockingLLGoFiles = "_wrap/blocking_unix.c"

// CondWait preserves the caller's roots without suspending the blocked thread
// during collection. It restores the active GC state before returning.
//
//go:linkname CondWait C.llgo_gc_cond_wait
func CondWait(cond, mutex unsafe.Pointer) c.Int
