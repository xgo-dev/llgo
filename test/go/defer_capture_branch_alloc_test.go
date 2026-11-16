//go:build !wasm && !baremetal && !nogc

package gotest

import (
	"runtime"
	"testing"
)

func TestDeferCaptureBranchAllocations(t *testing.T) {
	previous := branchDeferCallback
	defer func() { branchDeferCallback = previous }()
	branchDeferCallback = branchDeferStep(branchDeferInvoke)
	words := []uint64{7}
	// Warm metadata and defer machinery before measuring the retained flag.
	for i := 0; i < 64; i++ {
		branchDeferCapture(words, false)
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < 10000; i++ {
		branchDeferCapture(words, false)
	}
	runtime.ReadMemStats(&after)
	// Process-wide counters can include unrelated runtime activity. A retained
	// capture allocates once per call; allow only incidental allocations.
	if n := after.Mallocs - before.Mallocs; n > 16 {
		t.Fatalf("10000 frame-bounded defer calls allocated %d objects", n)
	}
}
