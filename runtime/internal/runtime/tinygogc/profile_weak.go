//go:build wasm && llgo.wasm.gc.linear

package tinygogc

import (
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
	"unsafe"
)

// ProfileWeakLink observes reclamation without changing finalizer ordering or
// retaining the profiled allocation. The profiler owns the node and slot.
type ProfileWeakLink struct {
	objectKey uintptr
	slot      *uintptr
	next      *ProfileWeakLink
}

var profileWeakLinks *ProfileWeakLink

func RegisterProfileWeakLink(link *ProfileWeakLink, ptr unsafe.Pointer, slot *uintptr) bool {
	lock(&gcMutex)
	address := uintptr(ptr)
	if !isOnHeap(address) || gcStateOf(blockFromAddr(address)) == blockStateFree {
		unlock(&gcMutex)
		return false
	}
	link.objectKey = ^gcAddressOf(gcFindHead(blockFromAddr(address)))
	link.slot = slot
	link.next = profileWeakLinks
	profileWeakLinks = link
	unlock(&gcMutex)
	return true
}

// The world is stopped and finalizable objects have been preserved. Clear and
// unlink dead nodes before addresses can be reused by the allocator.
func clearProfileWeakLinks() {
	link := &profileWeakLinks
	for *link != nil {
		node := *link
		block := blockFromAddr(^node.objectKey)
		if gcStateOf(block) == blockStateMark {
			link = &node.next
			continue
		}
		*link = node.next
		atomic.Store(node.slot, uintptr(0))
		node.slot = nil
		node.next = nil
	}
}

func ProfileObjectSize(requested uintptr) uintptr {
	return (requested + bytesPerBlock - 1) / bytesPerBlock * bytesPerBlock
}
