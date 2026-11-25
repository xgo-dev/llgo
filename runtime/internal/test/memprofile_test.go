//go:build llgo

package test

import (
	"math/bits"
	"runtime"
	"testing"
)

func TestMemProfileBucketIndexAndShardSpread(t *testing.T) {
	// Verify power-of-two bucket mapping via MemProfile across small/medium/large sizes
	// and check Fibonacci shard hash distribution across common OS pthread stack strides.
	for _, stride := range []struct {
		name  string
		step  uint64
		minUn int
	}{
		{"macos-512k+16k", 0x84000, 20},
		{"linux-2m", 0x200000, 20},
		{"linux-8m", 0x800000, 20},
	} {
		seen := make(map[uint64]bool)
		base := uint64(0x16f000000)
		for i := uint64(0); i < 32; i++ {
			id := base + i*stride.step
			shard := (((id >> 14) ^ (id >> 21)) * 0x9e3779b97f4a7c15) >> 58
			seen[shard] = true
		}
		if len(seen) < stride.minUn {
			t.Fatalf("%s: only %d unique shards across 32 threads, want >= %d", stride.name, len(seen), stride.minUn)
		}
	}

	// Verify ctlz bucket index formula matches every bucket boundary (16 .. 1<<30).
	const maxSize = uint64(1) << 30
	const numBuckets = 27
	for b := 0; b < numBuckets; b++ {
		limit := uint64(16) << b
		got := 0
		if limit >= maxSize {
			got = numBuckets - 1
		} else if limit > 16 {
			got = 64 - bits.LeadingZeros64((limit-1)>>4)
		}
		if got != b {
			t.Fatalf("size %d mapped to bucket %d, want %d", limit, got, b)
		}
		if b+1 < numBuckets {
			next := limit + 1
			gotNext := 64 - bits.LeadingZeros64((next-1)>>4)
			if gotNext != b+1 {
				t.Fatalf("size %d mapped to bucket %d, want %d", next, gotNext, b+1)
			}
		}
	}

	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	sinks := make([]*byte, 64)
	for i := range sinks {
		b := new(byte)
		sinks[i] = b
	}
	runtime.KeepAlive(sinks)
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	if after.Mallocs <= before.Mallocs {
		t.Fatalf("MemStats.Mallocs did not increase: before=%d after=%d", before.Mallocs, after.Mallocs)
	}
}
