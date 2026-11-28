package runtime

import (
	"sync/atomic"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/clite/tls"
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
	"github.com/xgo-dev/llgo/runtime/internal/wordmap"
)

// Share one TLS key to avoid exhausting the process-wide pthread key limit.
// The first poolFastSlotLimit sync.Pools receive sequential slot IDs for O(1)
// per-thread lookup. Allocate a page only when the thread touches a slot in it. Pools
// allocated past that one-way cap fall back to the sparse per-thread overflow
// wordmap (retaining entries until thread exit, matching the pre-existing map
// lifecycle).
const (
	poolFastSlotLimit = 1024
	poolLocalPageSize = 32
)

type poolLocalEntry struct {
	victim unsafe.Pointer
	local  unsafe.Pointer
}

type poolLocalPage [poolLocalPageSize]poolLocalEntry

type poolLocalTable struct {
	fast     [poolFastSlotLimit / poolLocalPageSize]*poolLocalPage
	overflow wordmap.Map[unsafe.Pointer, unsafe.Pointer]
}

type poolSlotMeta struct {
	id     uintptr
	victim unsafe.Pointer
}

var poolLocalOnce psync.Once
var poolLocals tls.Handle[*poolLocalTable]
var poolNextID atomic.Uintptr

func allocatePoolSlot(next *atomic.Uintptr) uintptr {
	for id := next.Load(); id < poolFastSlotLimit; id = next.Load() {
		if next.CompareAndSwap(id, id+1) {
			return id
		}
	}
	// Fast-slot owners may outlive the counter hitting its limit. Keep the
	// counter saturated so that no later pool can receive one of their IDs.
	return poolFastSlotLimit
}

//go:linkname syncRuntimePoolLocalAlloc sync.runtime_poolLocalAlloc
func syncRuntimePoolLocalAlloc(victim *unsafe.Pointer) unsafe.Pointer {
	poolLocalOnce.Do(func() {
		poolLocals = tls.AllocReusable[*poolLocalTable](func(locals **poolLocalTable) {
			if tbl := *locals; tbl != nil {
				for _, page := range tbl.fast {
					if page == nil {
						continue
					}
					for _, e := range page {
						if e.local != nil && e.victim != nil {
							atomic.StorePointer((*unsafe.Pointer)(e.victim), e.local)
						}
					}
				}
				tbl.overflow.Range(func(victimField, local unsafe.Pointer) {
					atomic.StorePointer((*unsafe.Pointer)(victimField), local)
				})
			}
		})
	})
	id := allocatePoolSlot(&poolNextID)
	// Retain the victim pointer inside the handle so the pool's victim field
	// stays reachable until thread TLS destructors publish cached locals.
	return unsafe.Pointer(&poolSlotMeta{id: id, victim: unsafe.Pointer(victim)})
}

//go:linkname syncRuntimePoolLocalGet sync.runtime_poolLocalGet
func syncRuntimePoolLocalGet(handle unsafe.Pointer) unsafe.Pointer {
	locals := poolLocals.Get()
	if locals == nil {
		return nil
	}
	meta := (*poolSlotMeta)(handle)
	if meta.id < poolFastSlotLimit {
		page := locals.fast[meta.id/poolLocalPageSize]
		if page == nil {
			return nil
		}
		return page[meta.id%poolLocalPageSize].local
	}
	return locals.overflow.Get(meta.victim)
}

//go:linkname syncRuntimePoolLocalSet sync.runtime_poolLocalSet
func syncRuntimePoolLocalSet(handle, local unsafe.Pointer) {
	meta := (*poolSlotMeta)(handle)
	locals := poolLocals.Get()
	if locals == nil {
		locals = &poolLocalTable{}
		poolLocals.Set(locals)
	}
	if meta.id < poolFastSlotLimit {
		pageIndex := meta.id / poolLocalPageSize
		page := locals.fast[pageIndex]
		if page == nil {
			page = new(poolLocalPage)
			locals.fast[pageIndex] = page
		}
		page[meta.id%poolLocalPageSize] = poolLocalEntry{victim: meta.victim, local: local}
		return
	}
	locals.overflow.Set(meta.victim, local)
}
