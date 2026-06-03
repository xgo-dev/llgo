package metrics

import (
	"math"
	"runtime"
	"unsafe"
)

var llgoMetricDefaultHistBuckets = []float64{0, math.Inf(1)}

// runtime_readMetrics replaces the GOROOT declaration that is normally
// implemented by package runtime. Keep the catalog and value layouts in this
// package, so LLGo only supplies the small subset of values it can observe.
func runtime_readMetrics(samplesp unsafe.Pointer, n int, _ int) {
	samples := unsafe.Slice((*Sample)(samplesp), n)
	mem := llgoReadMetricMemStats()

	for i := range samples {
		sample := &samples[i]
		kind, ok := llgoMetricKind(sample.Name)
		if !ok {
			sample.Value = Value{}
			continue
		}
		llgoSetMetricDefault(&sample.Value, kind)
		llgoSetRuntimeMetric(sample.Name, &sample.Value, mem)
	}
}

func llgoMetricKind(name string) (ValueKind, bool) {
	for _, desc := range allDesc {
		if desc.Name == name {
			return desc.Kind, true
		}
	}
	return KindBad, false
}

func llgoSetRuntimeMetric(name string, value *Value, mem runtime.MemStats) {
	switch name {
	case "/sched/gomaxprocs:threads", "/sched/threads/total:threads":
		llgoSetUint64(value, uint64(runtime.GOMAXPROCS(0)))
	case "/sched/goroutines-created:goroutines",
		"/sched/goroutines/running:goroutines",
		"/sched/goroutines:goroutines":
		llgoSetUint64(value, 1)
	case "/gc/cycles/automatic:gc-cycles":
		llgoSetUint64(value, llgoSaturatingSub(uint64(mem.NumGC), uint64(mem.NumForcedGC)))
	case "/gc/cycles/forced:gc-cycles":
		llgoSetUint64(value, uint64(mem.NumForcedGC))
	case "/gc/cycles/total:gc-cycles":
		llgoSetUint64(value, uint64(mem.NumGC))
	case "/gc/heap/allocs:bytes":
		llgoSetUint64(value, mem.TotalAlloc)
	case "/gc/heap/allocs:objects":
		llgoSetUint64(value, mem.Mallocs)
	case "/gc/heap/frees:bytes":
		llgoSetUint64(value, llgoSaturatingSub(mem.TotalAlloc, mem.Alloc))
	case "/gc/heap/frees:objects":
		llgoSetUint64(value, mem.Frees)
	case "/gc/heap/goal:bytes":
		llgoSetUint64(value, mem.NextGC)
	case "/gc/heap/live:bytes", "/memory/classes/heap/objects:bytes":
		llgoSetUint64(value, mem.HeapAlloc)
	case "/gc/heap/objects:objects":
		llgoSetUint64(value, mem.HeapObjects)
	case "/memory/classes/heap/free:bytes":
		llgoSetUint64(value, llgoSaturatingSub(mem.HeapIdle, mem.HeapReleased))
	case "/memory/classes/heap/released:bytes":
		llgoSetUint64(value, mem.HeapReleased)
	case "/memory/classes/heap/stacks:bytes":
		llgoSetUint64(value, mem.StackInuse)
	case "/memory/classes/heap/unused:bytes":
		llgoSetUint64(value, llgoSaturatingSub(mem.HeapInuse, mem.HeapAlloc))
	case "/memory/classes/metadata/mcache/free:bytes":
		llgoSetUint64(value, llgoSaturatingSub(mem.MCacheSys, mem.MCacheInuse))
	case "/memory/classes/metadata/mcache/inuse:bytes":
		llgoSetUint64(value, mem.MCacheInuse)
	case "/memory/classes/metadata/mspan/free:bytes":
		llgoSetUint64(value, llgoSaturatingSub(mem.MSpanSys, mem.MSpanInuse))
	case "/memory/classes/metadata/mspan/inuse:bytes":
		llgoSetUint64(value, mem.MSpanInuse)
	case "/memory/classes/metadata/other:bytes":
		llgoSetUint64(value, mem.GCSys)
	case "/memory/classes/os-stacks:bytes":
		llgoSetUint64(value, llgoSaturatingSub(mem.StackSys, mem.StackInuse))
	case "/memory/classes/other:bytes":
		llgoSetUint64(value, mem.OtherSys)
	case "/memory/classes/profiling/buckets:bytes":
		llgoSetUint64(value, mem.BuckHashSys)
	case "/memory/classes/total:bytes":
		llgoSetUint64(value, mem.Sys)
	}
}

func llgoSetMetricDefault(value *Value, kind ValueKind) {
	switch kind {
	case KindUint64:
		llgoSetUint64(value, 0)
	case KindFloat64:
		value.kind = KindFloat64
		value.scalar = math.Float64bits(0)
		value.pointer = nil
	case KindFloat64Histogram:
		llgoFloat64HistOrInit(value, llgoMetricDefaultHistBuckets)
	default:
		*value = Value{}
	}
}

func llgoSetUint64(value *Value, n uint64) {
	value.kind = KindUint64
	value.scalar = n
	value.pointer = nil
}

func llgoFloat64HistOrInit(value *Value, buckets []float64) *Float64Histogram {
	var hist *Float64Histogram
	if value.kind == KindFloat64Histogram && value.pointer != nil {
		hist = (*Float64Histogram)(value.pointer)
	} else {
		hist = new(Float64Histogram)
		value.pointer = unsafe.Pointer(hist)
	}
	value.kind = KindFloat64Histogram
	value.scalar = 0
	hist.Buckets = buckets
	if len(hist.Counts) != len(buckets)-1 {
		hist.Counts = make([]uint64, len(buckets)-1)
	} else {
		clear(hist.Counts)
	}
	return hist
}

func llgoSaturatingSub(a, b uint64) uint64 {
	if a < b {
		return 0
	}
	return a - b
}
