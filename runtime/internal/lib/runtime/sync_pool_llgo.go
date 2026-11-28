package runtime

import (
	"sync/atomic"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/clite/tls"
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
	"github.com/xgo-dev/llgo/runtime/internal/wordmap"
)

// Share one TLS key to avoid exhausting the process-wide pthread key limit.
// The first poolFastSlotLimit sync.Pools receive sequential slot IDs so
// per-thread Get/Put lookups index a flat slice in O(1) without hashing (each
// fast slice grows lazily up to poolFastSlotLimit entries per thread). Pools
// allocated past that one-way cap fall back to the sparse per-thread overflow
// wordmap (retaining entries until thread exit, matching the pre-existing map
// lifecycle).
const poolFastSlotLimit = 1024

type poolLocalEntry struct {
	victim unsafe.Pointer
	local  unsafe.Pointer
}

type poolLocalTable struct {
	fast     []poolLocalEntry
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
				for i := range tbl.fast {
					e := &tbl.fast[i]
					if e.local != nil && e.victim != nil {
						atomic.StorePointer((*unsafe.Pointer)(e.victim), e.local)
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
		if meta.id >= uintptr(len(locals.fast)) {
			return nil
		}
		return locals.fast[meta.id].local
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
		id := int(meta.id)
		if id >= len(locals.fast) {
			if id < cap(locals.fast) {
				locals.fast = locals.fast[:id+1]
			} else {
				newCap := id + 1
				if newCap < 8 {
					newCap = 8
				}
				if cap(locals.fast)*2 > newCap {
					newCap = cap(locals.fast) * 2
				}
				if newCap > poolFastSlotLimit {
					newCap = poolFastSlotLimit
				}
				grown := make([]poolLocalEntry, id+1, newCap)
				copy(grown, locals.fast)
				locals.fast = grown
			}
		}
		locals.fast[id] = poolLocalEntry{victim: meta.victim, local: local}
		return
	}
	locals.overflow.Set(meta.victim, local)
}
