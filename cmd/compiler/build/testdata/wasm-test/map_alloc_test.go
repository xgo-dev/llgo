//go:build llgo && llgo.wasm.gc.linear

package wasmtest

import "testing"

func TestMapKeyHashNoAlloc(t *testing.T) {
	testMapKeyHashNoAlloc(t, uint32(1<<31))
	testMapKeyHashNoAlloc(t, int32(-1<<31))
	testMapKeyHashNoAlloc(t, uint64(1<<40))
}

func testMapKeyHashNoAlloc[K ~uint32 | ~int32 | ~uint64](t *testing.T, highBit K) {
	// Multiple buckets force hashing. High bits exercise the full key on
	// both memory32 and memory64; single-bucket lookups can bypass hashing.
	m := make(map[K]K, 64)
	for i := 0; i < 64; i++ {
		key := K(i) | highBit
		m[key] = key
	}
	// Match the call boundary in this instrumented binary: pushing a caller
	// frame can allocate even when the map operations themselves do not.
	base := testing.AllocsPerRun(100, emptyAllocControl)
	i := 0
	allocs := testing.AllocsPerRun(4096, func() {
		key := K(i%64) | highBit
		i++
		if m[key] != key {
			t.Fatal("single-result lookup returned the wrong value")
		}
		value, ok := m[key]
		if !ok || value != key {
			t.Fatal("comma-ok lookup returned the wrong value")
		}
		m[key] = value
		delete(m, key)
		m[key] = key
	})
	if allocs != base {
		t.Fatalf("%T map operations added %v allocations per iteration", highBit, allocs-base)
	}
}
