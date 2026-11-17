//go:build !wasm && !baremetal && !nogc

package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/clite/bdwgc"
)

type heapProfileWeakLink struct{}

func allocHeapProfilePCs(size uintptr) unsafe.Pointer {
	p := bdwgc.MallocAtomic(size)
	if p == nil {
		panic("out of memory")
	}
	return p
}

func registerHeapProfileWeak(s *heapProfileSample, p unsafe.Pointer) bool {
	return bdwgc.RegisterLongLink(&s.key, bdwgc.Base(p)) == 0
}

// Report logical Go allocation bytes; collector headers and size-class slack
// must not change the sampling probability or pprof's inverse weighting.
func heapProfileObjectSize(_ unsafe.Pointer, requested uintptr) uintptr {
	if requested < 16 {
		return 16
	}
	return requested
}

func freeHeapProfilePCs(p unsafe.Pointer) { bdwgc.Free(p) }

func pinHeapProfilePCs(pcs []uintptr)                      {}
func heapProfileFrameForPC(pc uintptr) (CallerFrame, bool) { return CallerFrame{}, false }

// The creator walker already captures raw PCs using registered stack bounds.
// Delay symbol-index initialization until a profile is actually inspected.
func captureHeapProfilePCs(pcs []uintptr) int {
	if TracebackCreator != nil {
		return TracebackCreator(pcs)
	}
	return 0
}
