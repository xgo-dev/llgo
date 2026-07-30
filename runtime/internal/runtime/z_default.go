//go:build !baremetal

package runtime

import (
	"unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/clite/debug"
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
		if link == nil {
			node := (*panicNode)(ptr)
			TracePanic(node.arg)
			if PanicTraceback == nil || !PanicTraceback(2) {
				debug.PrintStack(2)
			}
			c.Free(unsafe.Pointer(node))
			c.Exit(2)
		} else {
			node := (*panicNode)(ptr)
			if link == gp.defer_ {
				node.defer_ = link
			}
			c.Siglongjmp(link.Addr, 1)
		}
	} else if gp.goexit {
		// Goexit must run deferred functions before terminating the current
		// goroutine. Reuse the longjmp-based defer unwinding:
		// 1) If we have a defer frame, longjmp to it so it can execute defers.
		// 2) Once we've unwound past the last frame (link==nil), terminate the
		//    current pthread.
		if link != nil {
			c.Siglongjmp(link.Addr, 1)
		}
		if gp.isMain {
			fatal("no goroutines (main called runtime.Goexit) - deadlock!")
			c.Exit(2)
		}
		leaveCurrentLocalContext()
		exitCurrentM()
	}
}
