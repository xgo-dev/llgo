//go:build llgo

package llgoext

import (
	"runtime"
	"testing"
)

var deferCaptureAllocSink int
var deferCaptureAllocRecords [64]runtime.MemProfileRecord

//go:noinline
func deferCaptureAllocEvents() uint64 {
	n, ok := runtime.MemProfile(deferCaptureAllocRecords[:], true)
	if !ok {
		panic("allocation profile capacity")
	}
	var total uint64
	for _, record := range deferCaptureAllocRecords[:n] {
		total += uint64(record.AllocObjects)
	}
	return total
}

//go:noinline
func deferCaptureAllocation(value int, callback func()) {
	returned := false
	defer func() {
		if !returned {
			panic("callback did not return")
		}
		deferCaptureAllocSink += value
	}()
	callback()
	returned = true
}

func TestDeferCaptureNoAllocation(t *testing.T) {
	callback := func() {}
	deferCaptureAllocation(0, callback)

	// These counters are process-wide: test timers or another runtime
	// thread may allocate during a measurement. Require a zero-allocation
	// run, allowing a few attempts to exclude that unrelated activity.
	best := ^uint64(0)
	var checksum int
	for attempt := 0; attempt < 4; attempt++ {
		deferCaptureAllocSink = 0
		before := deferCaptureAllocEvents()
		for i := 1; i <= 100; i++ {
			deferCaptureAllocation(i, callback)
		}
		after := deferCaptureAllocEvents()
		checksum = deferCaptureAllocSink
		if after < before {
			t.Fatal("allocation profile wrapped")
		}
		if delta := after - before; delta < best {
			best = delta
		}
		if best == 0 && checksum == 5050 {
			return
		}
	}
	t.Fatalf("100 calls: allocations=%d, checksum=%d; want 0, 5050", best, checksum)
}
