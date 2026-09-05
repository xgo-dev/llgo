//go:build wasm && llgo.wasm.gc.linear

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/debug"
	"github.com/xgo-dev/llgo/runtime/internal/gcroot"
)

// Rethrow rethrows a panic after discarding roots owned by skipped frames.
func Rethrow(link *Defer) {
	gp := getg()
	if ptr := gp.panic_; ptr != nil {
		if gp.panicIsSuspended(ptr) {
			return
		}
		node := (*panicNode)(ptr)
		gp.movePanicToDefer(node, link)
		if link == nil {
			TracePanic(node.arg)
			if PanicTraceback == nil || !PanicTraceback(2) {
				debug.PrintStack(2)
			}
			c.Free(unsafe.Pointer(node))
			c.Exit(2)
		} else {
			gcroot.BeginSJLJReplay()
			gcroot.RestoreChain(link.gcRoot)
			c.Siglongjmp(link.Addr, 1)
		}
	} else if gp.goexit {
		gp.defer_ = link
		if link != nil {
			gcroot.BeginSJLJReplay()
			gcroot.RestoreChain(link.gcRoot)
			c.Siglongjmp(link.Addr, 1)
		}
		if gp.isMain {
			markMainExited()
		}
		goexitBackend(gp)
	}
}
