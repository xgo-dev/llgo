//go:build !nogc && !baremetal

package runtime

import (
	"runtime"

	"github.com/goplus/llgo/runtime/internal/clite/bdwgc"
	llrt "github.com/goplus/llgo/runtime/internal/runtime"
)

func init() {
	bdwgc.Init()
}

func ReadMemStats(m *runtime.MemStats) {
	if m == nil {
		return
	}
	var heapSize, freeBytes, unmappedBytes, bytesSinceGC, totalBytes uintptr
	bdwgc.GetHeapUsageSafe(&heapSize, &freeBytes, &unmappedBytes, &bytesSinceGC, &totalBytes)

	heapSys := heapSize + unmappedBytes
	heapIdle := freeBytes + unmappedBytes
	heapInuse := saturatingSub(heapSys, heapIdle)
	heapAlloc := heapInuse
	*m = runtime.MemStats{
		Alloc:      uint64(heapAlloc),
		TotalAlloc: uint64(totalBytes),
		Sys:        uint64(heapSys),
		HeapAlloc:  uint64(heapAlloc),
		HeapSys:    uint64(heapSys),
		HeapIdle:   uint64(heapIdle),
		HeapInuse:  uint64(heapInuse),
		NumGC:      uint32(bdwgc.GetGCNo()),
	}
}

func GC() {
	llrt.SetMemProfileRate(MemProfileRate)
	bdwgc.Gcollect()
	// BDW finalizers are observed on a subsequent collection cycle.
	// Run one extra cycle so weak-pointer cleanup hooks (unique/weak) see
	// finalized state before we trigger map cleanup callbacks.
	bdwgc.Gcollect()
	unique_runtime_notifyMapCleanup()
	if poolCleanup != nil {
		poolCleanup()
	}
}

func saturatingSub(x, y uintptr) uintptr {
	if x < y {
		return 0
	}
	return x - y
}
