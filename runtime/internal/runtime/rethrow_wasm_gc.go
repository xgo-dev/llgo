//go:build wasm && llgo_wasm_gc

package runtime

import (
	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/clite/debug"
	"github.com/goplus/llgo/runtime/internal/gcroot"
)

// Rethrow rethrows a panic after discarding roots owned by skipped frames.
func Rethrow(link *Defer) {
	gp := getg()
	if ptr := gp.panic_; ptr != nil {
		if link == nil {
			TracePanic(*(*any)(ptr))
			if PanicTraceback == nil || !PanicTraceback(2) {
				debug.PrintStack(2)
			}
			c.Free(ptr)
			c.Exit(2)
		} else {
			gcroot.RestoreChain(link.gcRoot)
			c.Siglongjmp(link.Addr, 1)
		}
	} else if gp.goexit {
		if link != nil {
			gcroot.RestoreChain(link.gcRoot)
			c.Siglongjmp(link.Addr, 1)
		}
		goexitBackend(gp)
	}
}
