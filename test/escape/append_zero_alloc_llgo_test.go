//go:build llgo && !wasm && !baremetal && !nogc

package escape

import (
	"runtime"
	"testing"
)

var freshZeroAppendSink []byte

//go:noinline
func freshZeroAppendDirect(n int) []byte { return make([]byte, n) }

//go:noinline
func freshZeroAppendFolded(n int) []byte { return append([]byte(nil), make([]byte, n)...) }

func TestFreshZeroAppendAllocations(t *testing.T) {
	measure := func(build func(int) []byte) uint64 {
		for i := 0; i < 64; i++ {
			freshZeroAppendSink = build(4096)
		}
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		for i := 0; i < 1024; i++ {
			freshZeroAppendSink = build(4096)
			if len(freshZeroAppendSink) != 4096 || freshZeroAppendSink[0] != 0 || freshZeroAppendSink[4095] != 0 {
				t.Fatal("incorrect zero slice")
			}
		}
		runtime.ReadMemStats(&after)
		return after.TotalAlloc - before.TotalAlloc
	}
	direct, appended := measure(freshZeroAppendDirect), measure(freshZeroAppendFolded)
	t.Logf("1024 calls: direct=%d append=%d collector bytes", direct, appended)
	// LLGo's collector counters include size-class overhead and incidental
	// runtime work. Reject a second full backing array without assuming Go's
	// exact byte accounting or its supported Mallocs counter. The tolerance is
	// over all 1024 calls; a redundant 4096-byte array per call adds over 4 MiB.
	if appended > direct+(128<<10) {
		t.Fatalf("fresh append retained a redundant temporary: direct=%d append=%d", direct, appended)
	}
}
