package gotest

import (
	"runtime"
	"testing"
)

//go:noinline
func zeroAllocationBytes(n int) []byte { return make([]byte, n) }

type zeroAllocationRecord struct {
	p *int
	s string
	b []byte
	n uint64
}

//go:noinline
func zeroAllocationRecords(n int) []zeroAllocationRecord {
	return make([]zeroAllocationRecord, n)
}

func TestAllocationZeroed(t *testing.T) {
	// TestRuntimeAlignedAllocationZeroed/GC in aligned_alloc_llgo_test.go
	// additionally cover higher alignments, fallback storage, and bad requests.
	// Dirty objects and collect between rounds to exercise reused storage as
	// well as fresh allocations, including allocator size-class boundaries.
	for round := 0; round < 4; round++ {
		for _, size := range []int{0, 1, 7, 8, 15, 16, 17, 4095, 4096, 4097, 65537} {
			b := zeroAllocationBytes(size)
			if b == nil || len(b) != size {
				t.Fatalf("make([]byte, %d): incorrect slice", size)
			}
			for i, v := range b {
				if v != 0 {
					t.Fatalf("round %d size %d byte %d = %d", round, size, i, v)
				}
				b[i] = 0xff
			}
		}
		records := zeroAllocationRecords(37)
		for i, r := range records {
			if r.p != nil || r.s != "" || r.b != nil || r.n != 0 {
				t.Fatalf("round %d record %d is not zero", round, i)
			}
			records[i] = zeroAllocationRecord{new(int), "dirty", []byte{1}, ^uint64(0)}
		}
		runtime.GC()
		runtime.KeepAlive(records)
	}
}
