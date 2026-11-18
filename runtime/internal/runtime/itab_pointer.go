//go:build !baremetal

package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

func loadItabPointer(ptr *unsafe.Pointer) unsafe.Pointer {
	return atomic.Load(ptr)
}

func storeItabPointer(ptr *unsafe.Pointer, value unsafe.Pointer) {
	atomic.Store(ptr, value)
}
