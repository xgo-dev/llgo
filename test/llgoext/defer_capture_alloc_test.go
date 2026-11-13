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
	deferCaptureAllocSink = 0
	callback := func() {}
	deferCaptureAllocation(0, callback)
	before := deferCaptureAllocEvents()
	for i := 1; i <= 100; i++ {
		deferCaptureAllocation(i, callback)
	}
	after := deferCaptureAllocEvents()
	if after != before || deferCaptureAllocSink != 5050 {
		t.Fatalf("100 calls: allocations=%d, checksum=%d; want 0, 5050", after-before, deferCaptureAllocSink)
	}
}
