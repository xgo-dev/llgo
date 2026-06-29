//go:build baremetal

package runtime

import (
	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/clite/setjmp"
)

var (
	printFormatPrefixInt  = c.Str("%ld")
	printFormatPrefixUInt = c.Str("%lu")
	printFormatPrefixHex  = c.Str("%lx")
)

// Rethrow rethrows a panic.
// In baremetal single-threaded environment, we use longjmp to execute defers.
// Note: recover() will return nil for now (panic value not stored).
func Rethrow(link *Defer) {
	if link == nil {
		c.Printf(c.Str("fatal error\n"))
		c.Exit(2)
	} else {
		if ptr := panicKey.Get(); ptr != nil && link == GetThreadDefer() {
			node := (*panicNode)(ptr)
			node.defer_ = link
		}
		setjmp.Longjmp((*setjmp.JmpBuf)(link.Addr), 1)
	}
}
