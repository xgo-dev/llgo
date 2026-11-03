//go:build !baremetal && !wasm && !windows

package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/stacktrace"
)

// Capture only raw PCs, bounded by this thread's native stack. Unlike the
// public walker, this must not initialize the function/line tables. Native
// startup frames are harmless here: creatorPC filters them when formatting.
//
//go:noinline
func fpCreatorCallers(pcs []uintptr) int {
	var low, high uintptr
	stacktrace.Bounds(&low, &high)
	if high == 0 {
		// Preserve the symbol-bounded walk on an unregistered foreign thread.
		return fpCallers(0, pcs)
	}
	word := unsafe.Sizeof(uintptr(0))
	fp := uintptr(c_framepointer())
	n := 0
	for fp >= low && fp <= high-2*word && fp&(word-1) == 0 && n < len(pcs) {
		prev := *(*uintptr)(unsafe.Pointer(fp))
		pc := *(*uintptr)(unsafe.Pointer(fp + word))
		if pc < minLegalPC {
			break
		}
		pcs[n] = pc
		n++
		if prev <= fp || prev-fp > maxFPStride {
			break
		}
		fp = prev
	}
	return n
}
