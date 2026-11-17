//go:build wasm && llgo.wasm.gc.linear && !nogc && !baremetal

package runtime

import (
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/runtime/tinygogc"
	"unsafe"
)

type heapProfileWeakLink = tinygogc.ProfileWeakLink

// Numeric PCs must stay outside the conservatively scanned managed heap.
// Bucket arrays persist; per-G scratch is released with the G context.
func allocHeapProfilePCs(size uintptr) unsafe.Pointer {
	p := c.Malloc(size)
	if p == nil {
		panic("out of memory")
	}
	return p
}

func freeHeapProfilePCs(p unsafe.Pointer) { c.Free(p) }

func registerHeapProfileWeak(s *heapProfileSample, p unsafe.Pointer) bool {
	return tinygogc.RegisterProfileWeakLink(&s.weak, p, &s.key)
}

func heapProfileObjectSize(_ unsafe.Pointer, requested uintptr) uintptr {
	return tinygogc.ProfileObjectSize(requested)
}

func captureHeapProfilePCs(pcs []uintptr) int { return Callers(0, pcs) }
