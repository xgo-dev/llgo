//go:build baremetal

package runtime

import (
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/setjmp"
)

var (
	printFormatPrefixInt  = c.Str("%ld")
	printFormatPrefixUInt = c.Str("%lu")
	printFormatPrefixHex  = c.Str("%lx")
)

// Rethrow rethrows a panic.
// In baremetal single-threaded environment, we use longjmp to execute defers.
func Rethrow(link *Defer) {
	gp := getg()
	if ptr := gp.panic_; gp.panicIsSuspended(ptr) {
		return
	}
	if ptr := gp.panic_; ptr != nil {
		gp.movePanicToDefer((*panicNode)(ptr), link)
	} else {
		gp.defer_ = link
	}
	if link == nil {
		// Bare-metal has one execution context and no goroutine-exit
		// transition. Goexit still drains every defer through the longjmp
		// path, then retains the backend's existing fatal termination here.
		c.Printf(c.Str("fatal error\n"))
		c.Exit(2)
	} else {
		setjmp.Longjmp((*setjmp.JmpBuf)(link.Addr), 1)
	}
}
