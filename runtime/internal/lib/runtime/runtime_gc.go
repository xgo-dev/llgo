//go:build !nogc && !baremetal && !wasm

package runtime

import (
	"runtime"

	"github.com/xgo-dev/llgo/runtime/internal/clite/bdwgc"
	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

func init() {
	bdwgc.Init()
	enableForeignThreadRegistration()
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
	// Byte totals use BDWGC accounting; object events use the Go allocator's
	// existing counters. BDWGC does not expose collected/live object counts,
	// so Frees and HeapObjects remain zero, including in runtime/metrics.
	*m = runtime.MemStats{
		Alloc:      uint64(heapAlloc),
		TotalAlloc: uint64(totalBytes),
		Sys:        uint64(heapSys),
		Mallocs:    llruntime.MemProfileMallocs(),
		HeapAlloc:  uint64(heapAlloc),
		HeapSys:    uint64(heapSys),
		HeapIdle:   uint64(heapIdle),
		HeapInuse:  uint64(heapInuse),
		NumGC:      uint32(bdwgc.GetGCNo()),
	}
}

func GC() {
	requestExplicitGC()
	unique_runtime_notifyMapCleanup()
	if poolCleanup != nil {
		poolCleanup()
	}
}

func collectExplicitGCBatch() {
	collectAndEnqueueFinalizers()
	// Run one extra cycle for collector-side weak-pointer cleanup before map
	// cleanup callbacks. User finalizers run independently of the GC caller.
	collectAndEnqueueFinalizers()
}

func collectAndEnqueueFinalizers() {
	bdwgc.Gcollect()
	// Explicitly drain BDWGC's ready queue, including when a C library enables
	// finalize-on-demand. Go finalizers are enqueued for the finalizer worker;
	// foreign C finalizers may execute here or inside GC_gcollect and reenter GC.
	bdwgc.InvokeFinalizers()
	// Let mutators run before another stop-the-world cycle, including the
	// second collection performed by the same runtime.GC call.
	llruntime.Gosched()
}

func saturatingSub(x, y uintptr) uintptr {
	if x < y {
		return 0
	}
	return x - y
}
