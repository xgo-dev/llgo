package runtime

import (
	"sync/atomic"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/clite/tls"
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
	"github.com/xgo-dev/llgo/runtime/internal/wordmap"
)

// Share one TLS key to avoid exhausting the process-wide pthread key limit.
// Thread exit releases slot values; the key itself remains allocated.
// Each thread retains the pools it has used and their locals until exit, as
// the previous per-pool TLS slots did through their destructor closures.
type poolLocalTable = wordmap.Map[unsafe.Pointer, unsafe.Pointer]

var poolLocalOnce psync.Once
var poolLocals tls.Handle[*poolLocalTable]

//go:linkname syncRuntimePoolLocalAlloc sync.runtime_poolLocalAlloc
func syncRuntimePoolLocalAlloc(victim *unsafe.Pointer) unsafe.Pointer {
	poolLocalOnce.Do(func() {
		poolLocals = tls.Alloc[*poolLocalTable](func(locals **poolLocalTable) {
			if *locals != nil {
				(*locals).Range(func(victimField, local unsafe.Pointer) {
					atomic.StorePointer((*unsafe.Pointer)(victimField), local)
				})
			}
		})
	})
	// The victim field identifies the pool and keeps its owner alive until
	// this thread's locals have been published by the TLS destructor.
	return unsafe.Pointer(victim)
}

//go:linkname syncRuntimePoolLocalGet sync.runtime_poolLocalGet
func syncRuntimePoolLocalGet(handle unsafe.Pointer) unsafe.Pointer {
	locals := poolLocals.Get()
	if locals == nil {
		return nil
	}
	return locals.Get(handle)
}

//go:linkname syncRuntimePoolLocalSet sync.runtime_poolLocalSet
func syncRuntimePoolLocalSet(handle, local unsafe.Pointer) {
	locals := poolLocals.Get()
	if locals == nil {
		locals = new(poolLocalTable)
		poolLocals.Set(locals)
	}
	locals.Set(handle, local)
}
