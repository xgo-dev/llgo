package gotest

import (
	"runtime"
	"testing"
)

func TestMapRangeRepeatedInitialization(t *testing.T) {
	// The same range instruction must clear its previous hiter even when the
	// next map is nil or empty and mapiterinit returns without filling it.
	maps := []map[int]int{{1: 11, 2: 22}, nil, {}, {3: 33}, nil, {4: 44}}
	for round := 0; round < 3; round++ {
		sum, count := 0, 0
		for _, m := range maps {
			for k, v := range m {
				if v != 11*k {
					t.Fatalf("stale iterator entry: %d=%d", k, v)
				}
				sum += v
				count++
			}
		}
		if sum != 110 || count != 4 {
			t.Fatalf("sum=%d count=%d", sum, count)
		}
	}
}

func TestMapRangeNestedIterators(t *testing.T) {
	m := map[int]int{1: 10, 2: 20, 3: 30}
	pairs, sum := 0, 0
	for k, v := range m {
		for other, value := range m {
			if other == k {
				continue
			}
			pairs++
			sum += v + value
		}
	}
	if pairs != 6 || sum != 240 {
		t.Fatalf("nested iteration: pairs=%d sum=%d", pairs, sum)
	}
}

//go:noinline
func mapRangePointerValues() map[int]*[1024]byte {
	m := make(map[int]*[1024]byte)
	for i := 0; i < 32; i++ {
		p := new([1024]byte)
		p[0], p[1023] = byte(i), byte(i+1)
		m[i] = p
	}
	return m
}

func TestMapRangeKeepsPointersAcrossGC(t *testing.T) {
	// Only the iterator retains the returned map while the loop calls GC.
	seen := uint32(0)
	for k, p := range mapRangePointerValues() {
		if k%8 == 0 {
			runtime.GC()
		}
		if k < 0 || k >= 32 || p[0] != byte(k) || p[1023] != byte(k+1) {
			t.Fatalf("collected map entry: %d %v", k, p)
		}
		seen |= uint32(1) << uint(k)
	}
	if seen != ^uint32(0) {
		t.Fatalf("missing entries after GC: %08x", seen)
	}
}

func TestMapRangeGrowAcrossGC(t *testing.T) {
	// Pointer-free maps keep overflow buckets alive through iterator-owned
	// pointers, including buckets from earlier generations after a grow.
	m := make(map[int]int)
	for i := 0; i < 64; i++ {
		m[i] = i + 1
	}
	var seen [64]bool
	first := true
	for k, v := range m {
		if first {
			for i := 64; i < 1024; i++ {
				m[i] = i + 1
			}
			runtime.GC()
			first = false
		}
		if v != k+1 {
			t.Fatalf("entry changed after grow and GC: %d=%d", k, v)
		}
		if k >= 0 && k < len(seen) {
			if seen[k] {
				t.Fatalf("original entry yielded twice: %d", k)
			}
			seen[k] = true
		}
	}
	// Newly inserted entries may be yielded or skipped. Original entries
	// were never deleted and must all be yielded exactly once.
	for k, ok := range seen {
		if !ok {
			t.Fatalf("original entry lost after grow and GC: %d", k)
		}
	}
}
