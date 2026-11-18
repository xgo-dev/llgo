//go:build llgo && !baremetal && !wasm && !nogc

package metrics_test

import (
	"runtime"
	"runtime/metrics"
	"testing"
)

func TestBDWGCObjectMetrics(t *testing.T) {
	// BDWGC reports allocation events, but not live/free object counts.
	// The live metric must not silently become a cumulative allocation count.
	runtime.GC()
	samples := []metrics.Sample{
		{Name: "/gc/heap/allocs:objects"},
		{Name: "/gc/heap/frees:objects"},
		{Name: "/gc/heap/objects:objects"},
	}
	metrics.Read(samples)
	if samples[0].Value.Uint64() == 0 {
		t.Fatal("allocation metric reports no events")
	}
	for _, sample := range samples[1:] {
		if got := sample.Value.Uint64(); got != 0 {
			t.Fatalf("unavailable %s = %d, want 0", sample.Name, got)
		}
	}
}
