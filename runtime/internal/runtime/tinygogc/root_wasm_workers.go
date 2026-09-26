//go:build llgo && js && wasm && llgo.wasm.gc.linear && llgo.wasm.workers

package tinygogc

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// rootAllocations keeps explicitly rooted allocations reachable until
// FreeRoot removes them. The intrusive links live in the allocations, so
// publishing a root does not require another potentially collecting
// allocation. This provides the same scanned, uncollectable lifetime as
// BDWGC's GC_malloc_uncollectable for scheduler records that cross host
// boundaries.
var rootAllocations *rootAllocation

type rootAllocation struct {
	prev *rootAllocation
	next *rootAllocation
}

func AllocRoot(size uintptr) unsafe.Pointer {
	headerSize := unsafe.Sizeof(rootAllocation{})
	if size > ^uintptr(0)-headerSize {
		gcPanic(c.Str("gc: root allocation size overflow"))
	}
	root := (*rootAllocation)(Alloc(headerSize + size))
	lock(&gcMutex)
	root.next = rootAllocations
	if root.next != nil {
		root.next.prev = root
	}
	rootAllocations = root
	unlock(&gcMutex)
	return unsafe.Add(unsafe.Pointer(root), headerSize)
}

// FreeRoot releases the explicit root. The normal sweep reclaims its backing
// allocation after it becomes otherwise unreachable.
func FreeRoot(ptr unsafe.Pointer) {
	if ptr == nil {
		return
	}
	headerSize := unsafe.Sizeof(rootAllocation{})
	root := (*rootAllocation)(unsafe.Add(ptr, -int(headerSize)))
	lock(&gcMutex)
	if root.prev == nil {
		if rootAllocations != root {
			unlock(&gcMutex)
			gcPanic(c.Str("gc: invalid root allocation release"))
		}
		rootAllocations = root.next
	} else {
		root.prev.next = root.next
	}
	if root.next != nil {
		root.next.prev = root.prev
	}
	root.prev = nil
	root.next = nil
	unlock(&gcMutex)
}
