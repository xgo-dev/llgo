//go:build !baremetal && (!wasm || !llgo_wasm_gc)

package runtime

import (
	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/clite/debug"
)

// Rethrow rethrows a panic.
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
			c.Siglongjmp(link.Addr, 1)
		}
	} else if gp.goexit {
		// Goexit runs deferred functions before the selected scheduler removes
		// the current goroutine.
		if link != nil {
			c.Siglongjmp(link.Addr, 1)
		}
		goexitBackend(gp)
	}
}
