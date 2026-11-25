//go:build llgo && llgo.wasm.gc.linear

package wasmtest

import "testing"

type allocObject struct{ value int }

var unusedAllocSink *allocObject

//go:noinline
func emptyAllocControl() {}

//go:noinline
func unusedAlloc() { value := &allocObject{}; _ = value }

//go:noinline
func escapingAlloc() { unusedAllocSink = &allocObject{} }

func TestUnusedAllocation(t *testing.T) {
	// This test binary also enables caller-frame instrumentation. Compare
	// identical call boundaries so its allocation is not charged to the literal.
	base := testing.AllocsPerRun(100, emptyAllocControl)
	if n := testing.AllocsPerRun(100, unusedAlloc); n != base {
		t.Fatalf("unused composite literal added %v allocations, want 0", n-base)
	}
	if n := testing.AllocsPerRun(100, escapingAlloc); n != base+1 {
		t.Fatalf("escaping composite literal added %v allocations, want 1", n-base)
	}
}
