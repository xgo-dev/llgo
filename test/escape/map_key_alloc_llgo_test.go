//go:build llgo && !wasm && !baremetal && !nogc

package escape

import (
	"runtime"
	"testing"
)

func TestMapKeyTemporaryAllocations(t *testing.T) {
	type key struct{ A, B uint64 }
	k, missing := key{1, 2}, key{3, 4}
	m := map[key]int{k: 0}
	exercise := func(n int) {
		for i := 0; i < n; i++ {
			mapKeyUpdate(m, k, i)
			if got, ok := mapKeyLookup(m, k); !ok || got != i {
				t.Fatalf("lookup: %d, %v, want %d", got, ok, i)
			}
			mapKeyDelete(m, missing)
		}
	}
	exercise(2000)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	exercise(10000)
	runtime.ReadMemStats(&after)
	allocated := after.TotalAlloc - before.TotalAlloc
	t.Logf("10000 map update/lookup/delete rounds: %d collector bytes", allocated)
	// Collector accounting can include incidental runtime work. This bound
	// rejects a recurring key allocation without requiring exact Go counters.
	if allocated > 64<<10 {
		t.Fatalf("temporary keys allocated %d bytes", allocated)
	}
}

func TestRangeTemporaryAllocations(t *testing.T) {
	m := map[int]int{1: 41}
	exercise := func(n int) {
		for i := 0; i < n; i++ {
			if got := mapIterationSum(m); got != 42 {
				t.Fatalf("map range: %d, want 42", got)
			}
			if got := stringIterationSum("a世z"); got != 20209 {
				t.Fatalf("string range: %d, want 20209", got)
			}
		}
	}
	exercise(2000)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	exercise(10000)
	runtime.ReadMemStats(&after)
	allocated := after.TotalAlloc - before.TotalAlloc
	t.Logf("10000 map/string range rounds: %d collector bytes", allocated)
	if allocated > 64<<10 {
		t.Fatalf("temporary iterators allocated %d bytes", allocated)
	}
}
