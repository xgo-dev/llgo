//go:build (!llgo || !windows || nogc || baremetal) && !llgo_noffi

package reflect

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/ffi"
)

// libffi invokes the returned function directly, so platforms that do not
// need foreign-thread registration pay no wrapper-call or defer cost.
func makeFuncCallback(nout int) func(*ffi.Signature, unsafe.Pointer, *unsafe.Pointer, unsafe.Pointer) {
	switch nout {
	case 0:
		return bind0
	case 1:
		return bind1
	default:
		return bindn
	}
}
