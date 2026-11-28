package gotest

import (
	"runtime"
	"sync"
	"testing"
)

func checkZeroBytes(t *testing.T, buf []byte) {
	t.Helper()
	for i, b := range buf {
		if b != 0 {
			t.Fatalf("non-zero byte at offset %d (len=%d): got 0x%x", i, len(buf), b)
		}
	}
}

func TestAllocAtomicZero(t *testing.T) {
	sizes := []int{0, 1, 4096, 64 << 10, 70000, 128 << 10, 1114112}

	// Sparse writes followed by GC and reallocation.
	for round := range 4 {
		for _, n := range sizes {
			buf := make([]byte, n)
			checkZeroBytes(t, buf)
			if n > 0 {
				buf[0] = byte(round + 1)
				buf[n/2] = byte(round + 2)
				buf[n-1] = byte(round + 3)
			}
		}
		runtime.GC()
	}

	// Dense writes followed by GC and reallocation verify reused dirty blocks
	// are completely zeroed, including unaligned head and tail edges.
	for _, n := range []int{70000, 128 << 10, 1114112} {
		for range 3 {
			buf := make([]byte, n)
			checkZeroBytes(t, buf)
			for i := range buf {
				buf[i] = 0xA5
			}
			buf = nil
			runtime.GC()
		}
	}

	// Concurrent large pointer-free zero allocations across goroutines.
	var wg sync.WaitGroup
	for g := range 4 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := range 8 {
				n := sizes[3+(i+id)%4]
				buf := make([]byte, n)
				for off, b := range buf {
					if b != 0 {
						t.Errorf("goroutine %d iter %d: non-zero byte at %d (len=%d): 0x%x", id, i, off, n, b)
						return
					}
				}
				buf[0] = byte(id + 1)
				buf[n-1] = byte(i + 1)
				if i%3 == 0 {
					for j := 0; j < n; j += 4096 {
						buf[j] = 0x5A
					}
				}
			}
		}(g)
	}
	wg.Wait()
}
