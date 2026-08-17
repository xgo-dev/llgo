//go:build !baremetal

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/debug"
)

var (
	printFormatPrefixInt  = c.Str("%lld")
	printFormatPrefixUInt = c.Str("%llu")
	printFormatPrefixHex  = c.Str("%llx")
)

// Rethrow rethrows a panic.
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
			c.Siglongjmp(link.Addr, 1)
		}
	} else if gp.goexit {
		// Goexit must run deferred functions before terminating the current
		// goroutine. Reuse the longjmp-based defer unwinding:
		// 1) If we have a defer frame, longjmp to it so it can execute defers.
		// 2) Once we've unwound past the last frame (link==nil), terminate the
		//    current pthread.
		gp.defer_ = link
		if link != nil {
			c.Siglongjmp(link.Addr, 1)
		}
		if gp.isMain {
			markMainExited()
		}
		leaveCurrentLocalContext()
		exitCurrentM()
	}
}
