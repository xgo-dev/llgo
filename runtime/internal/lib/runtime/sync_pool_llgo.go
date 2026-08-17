package runtime

import (
	"sync/atomic"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/clite/tls"
)

//go:linkname syncRuntimePoolLocalAlloc sync.runtime_poolLocalAlloc
func syncRuntimePoolLocalAlloc(victim *unsafe.Pointer) unsafe.Pointer {
	handle := tls.Alloc[unsafe.Pointer](func(local *unsafe.Pointer) {
		if local != nil {
			atomic.StorePointer(victim, *local)
		}
	})
	return unsafe.Pointer(&handle)
}

//go:linkname syncRuntimePoolLocalGet sync.runtime_poolLocalGet
func syncRuntimePoolLocalGet(handle unsafe.Pointer) unsafe.Pointer {
	return (*tls.Handle[unsafe.Pointer])(handle).Get()
}

//go:linkname syncRuntimePoolLocalSet sync.runtime_poolLocalSet
func syncRuntimePoolLocalSet(handle, local unsafe.Pointer) {
	(*tls.Handle[unsafe.Pointer])(handle).Set(local)
}
