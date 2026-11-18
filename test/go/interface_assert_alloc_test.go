//go:build llgo && !nogc && !baremetal && !wasm

package gotest

import (
	"runtime"
	"testing"
)

func TestInterfaceAssertMethodTableAllocation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value assertMethodBase
		want  int
		ok    bool
	}{
		{"two-methods", &assertMethodPair{7}, 7, true},
		{"eight-methods", &assertMethodMany{assertMethodPair{9}}, 9, true},
		{"missing-method", &assertMethodMissing{}, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Populate the itab cache before measuring repeated checks.
			for i := 0; i < 64; i++ {
				checkAssertMethods(tc.value)
			}
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			const calls = 10000
			var sum, successes int
			for i := 0; i < calls; i++ {
				v, ok := checkAssertMethods(tc.value)
				sum += v
				if ok {
					successes++
				}
			}
			runtime.ReadMemStats(&after)
			wantSuccesses := 0
			if tc.ok {
				wantSuccesses = calls
			}
			if sum != calls*tc.want || successes != wantSuccesses {
				t.Fatalf("incorrect repeated assertion results: sum=%d successes=%d", sum, successes)
			}
			// BDWGC reports approximate allocated bytes, not allocation-event
			// counts. Allow incidental runtime work, but reject a per-check
			// method-table copy, even for the single-method failure case.
			if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 32<<10 {
				t.Fatalf("%d assertions allocated %d bytes; method checks must not copy the method table", calls, allocated)
			}
			runtime.KeepAlive(tc.value)
		})
	}
}
