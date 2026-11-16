//go:build windows && (amd64 || arm64)

package gotest

import (
	"runtime"
	"testing"
)

//go:noinline
func windowsCallersAllocationProbe(pcs []uintptr) int {
	return runtime.Callers(0, pcs)
}

func TestRuntimeCallersWindowsAllocations(t *testing.T) {
	pcs := make([]uintptr, 16)
	for range 128 {
		if windowsCallersAllocationProbe(pcs) < 2 {
			t.Fatal("missing caller frames")
		}
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range 4096 {
		if windowsCallersAllocationProbe(pcs) < 2 {
			t.Fatal("missing caller frames")
		}
	}
	runtime.ReadMemStats(&after)
	// Allow first-use symbol-cache entries for the measurement's call site,
	// but reject a new CONTEXT or per-frame output word on every unwind.
	if n := after.TotalAlloc - before.TotalAlloc; n > 128<<10 {
		t.Fatalf("4096 caller walks allocated %d bytes", n)
	}
}
