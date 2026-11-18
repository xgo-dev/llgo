//go:build !nogc

package testing_test

import (
	"runtime"
	"testing"
)

var allocationEventSink []byte
var allocationEventPointerSink *allocationEventNode

type allocationEventNode struct {
	next *allocationEventNode
	data int
}

//go:noinline
func allocateEventControl(size int) {
	allocationEventSink = make([]byte, size)
}

//go:noinline
func allocatePointerEventControl() {
	allocationEventPointerSink = &allocationEventNode{next: allocationEventPointerSink, data: 42}
}

func TestAllocationEventControls(t *testing.T) {
	for _, size := range []int{1, 16, 1024, 65536} {
		if n := testing.AllocsPerRun(100, func() { allocateEventControl(size) }); n < 1 {
			t.Fatalf("escaping %d-byte allocation reports %v allocations, want >= 1", size, n)
		}
	}
	if n := testing.AllocsPerRun(100, allocatePointerEventControl); n < 1 {
		t.Fatalf("escaping pointer-bearing object reports %v allocations, want >= 1", n)
	}
	if n := testing.AllocsPerRun(100, func() { allocationEventSink[0]++ }); n != 0 {
		t.Fatalf("in-place mutation reports %v allocations, want 0", n)
	}
	if n := testing.AllocsPerRun(100, func() { allocateEventControl(0) }); n != 0 {
		t.Fatalf("zero-sized allocation reports %v allocations, want 0", n)
	}
	// Keep snapshot storage outside the measured callback so this checks
	// ReadMemStats itself, independently of caller-side placement decisions.
	var stats runtime.MemStats
	if n := testing.AllocsPerRun(100, func() {
		runtime.ReadMemStats(&stats)
	}); n != 0 {
		t.Fatalf("ReadMemStats reports %v allocations, want 0", n)
	}
}

func TestReadMemStatsAllocationEvents(t *testing.T) {
	allocateEventControl(1024)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < 1000; i++ {
		allocateEventControl(1024)
	}
	runtime.ReadMemStats(&after)
	if count := after.Mallocs - before.Mallocs; count < 1000 {
		t.Fatalf("1000 escaping allocations report %d events, want >= 1000", count)
	}
	if after.TotalAlloc <= before.TotalAlloc {
		t.Fatal("escaping allocations did not increase TotalAlloc")
	}
	runtime.KeepAlive(allocationEventSink)
}

func TestBenchmarkAllocationEvents(t *testing.T) {
	result := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			allocateEventControl(1024)
		}
	})
	if result.AllocsPerOp() < 1 || result.AllocedBytesPerOp() == 0 {
		t.Fatalf("escaping benchmark allocation reports %s", result.MemString())
	}
}
