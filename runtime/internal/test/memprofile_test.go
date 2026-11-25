//go:build llgo

package test

import (
	"runtime"
	"testing"
	"unsafe"
)

//go:linkname rtMemProfileBucketIndex github.com/xgo-dev/llgo/runtime/internal/runtime.memProfileBucketIndex
func rtMemProfileBucketIndex(uintptr) int

//go:linkname rtMemProfileSizeClass github.com/xgo-dev/llgo/runtime/internal/runtime.memProfileSizeClass
func rtMemProfileSizeClass(uintptr) uintptr

func TestMemProfileBucketIndexAndBoundaries(t *testing.T) {
	const maxSize = uintptr(1) << 30
	const numBuckets = 27

	if got := rtMemProfileBucketIndex(0); got != 0 {
		t.Fatalf("size 0 mapped to bucket %d, want 0", got)
	}
	if got := rtMemProfileBucketIndex(16); got != 0 {
		t.Fatalf("size 16 mapped to bucket %d, want 0", got)
	}
	for b := 0; b < numBuckets; b++ {
		limit := uintptr(16) << b
		if got := rtMemProfileBucketIndex(limit); got != b {
			t.Fatalf("size %d mapped to bucket %d, want %d", limit, got, b)
		}
		if got := rtMemProfileSizeClass(limit); got != limit {
			t.Fatalf("size %d mapped to class %d, want %d", limit, got, limit)
		}
		if b > 0 {
			if got := rtMemProfileBucketIndex(limit - 1); got != b {
				t.Fatalf("size %d mapped to bucket %d, want %d", limit-1, got, b)
			}
		}
		if b+1 < numBuckets {
			if got := rtMemProfileBucketIndex(limit + 1); got != b+1 {
				t.Fatalf("size %d mapped to bucket %d, want %d", limit+1, got, b+1)
			}
		}
	}
	// Verify clamping at and above memProfileMaxSizeClass on both 32-bit and 64-bit.
	for _, size := range []uintptr{maxSize - 1, maxSize, maxSize + 1, ^uintptr(0)} {
		if got := rtMemProfileBucketIndex(size); got != numBuckets-1 {
			t.Fatalf("size %d mapped to bucket %d, want %d", size, got, numBuckets-1)
		}
		if got := rtMemProfileSizeClass(size); got != maxSize {
			t.Fatalf("size %d mapped to class %d, want %d", size, got, maxSize)
		}
	}
	if unsafe.Sizeof(uintptr(0)) == 8 {
		if got := rtMemProfileBucketIndex(maxSize << 2); got != numBuckets-1 {
			t.Fatalf("size %d mapped to bucket %d, want %d", maxSize<<2, got, numBuckets-1)
		}
	}

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	sinks := make([]*[64]byte, 64)
	for i := range sinks {
		p := new([64]byte)
		p[0] = byte(i)
		sinks[i] = p
	}
	runtime.KeepAlive(sinks)
	runtime.ReadMemStats(&after)
	if after.Mallocs < before.Mallocs+64 {
		t.Fatalf("MemStats.Mallocs delta = %d, want >= 64 (before=%d after=%d)", after.Mallocs-before.Mallocs, before.Mallocs, after.Mallocs)
	}
}
