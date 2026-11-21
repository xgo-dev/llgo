//go:build baremetal

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/setjmp"
)

// Rethrow rethrows a panic.
// In baremetal single-threaded environment, we use longjmp to execute defers.
func Rethrow(link *Defer) {
	gp := getg()
	if ptr := gp.panic_; gp.panicIsSuspended(ptr) {
		return
	}
	if ptr := gp.panic_; ptr != nil {
		node := (*panicNode)(ptr)
		gp.movePanicToDefer(node, link)
		if link == nil {
			TracePanic(node.arg)
			FreeRoot(unsafe.Pointer(node))
			c.Printf(c.Str("fatal error\n"))
			c.Exit(2)
		}
		setjmp.Longjmp((*setjmp.JmpBuf)(link.Addr), 1)
	}
	if gp.goexit {
		gp.defer_ = link
		if link != nil {
			setjmp.Longjmp((*setjmp.JmpBuf)(link.Addr), 1)
		}
		// Every deferred function has run; the selected backend terminates
		// this G (and, when it is main and no other G remains, the program).
		goexitBaremetal(gp)
	}
	// The deferred call recovered the panic and the unwound activation is
	// gone: returning lets the caller's recovery block resume normal
	// execution, matching the hosted backend.
}
