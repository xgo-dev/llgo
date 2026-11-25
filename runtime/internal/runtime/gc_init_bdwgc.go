//go:build !nogc && !baremetal && !wasm

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/bdwgc"
)

func cStrView(p *c.Char) string {
	if p == nil {
		return ""
	}
	n := c.Strlen(p)
	if n == 0 {
		return ""
	}
	return unsafe.String((*byte)(unsafe.Pointer(p)), n)
}

// InitGC configures BDWGC before runtime/internal/runtime.init runs so that
// marker thread counts take effect prior to the first GC_init().
//
// It runs before getg() or any package initializer; it must remain
// allocation-free and avoid calling getg().
func InitGC() {
	if bdwgc.IsInitCalled() != 0 {
		return
	}
	gomaxprocs := cStrView(cpuGetenv(c.Str("GOMAXPROCS")))
	gcMarkersSet := cpuGetenv(c.Str("GC_MARKERS")) != nil
	gcNprocs := cStrView(cpuGetenv(c.Str("GC_NPROCS")))
	if markers := targetBDWGCMarkers(gomaxprocs, gcMarkersSet, gcNprocs, CPUCount()); markers > 0 {
		bdwgc.SetMarkersCount(uint32(markers))
	}
}
