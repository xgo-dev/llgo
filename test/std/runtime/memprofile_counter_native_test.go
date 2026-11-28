//go:build llgo && !baremetal && !wasm

package runtime_test

import (
	"runtime"
	"sync"
	"testing"
	"unsafe"
)

//go:linkname nativeProfileSizeClass github.com/xgo-dev/llgo/runtime/internal/runtime.memProfileSizeClass
func nativeProfileSizeClass(uintptr) uintptr

//go:linkname nativeProfileBucketIndex github.com/xgo-dev/llgo/runtime/internal/runtime.memProfileBucketIndex
func nativeProfileBucketIndex(uintptr) int

//go:linkname nativeProfileShardIndex github.com/xgo-dev/llgo/runtime/internal/runtime.memProfileShardIndex
func nativeProfileShardIndex(uint64) uint64

//go:linkname nativeProfileAddBucket github.com/xgo-dev/llgo/runtime/internal/runtime.memProfileAddBucket
func nativeProfileAddBucket(int)

//go:linkname nativeProfileLoadBucket github.com/xgo-dev/llgo/runtime/internal/runtime.memProfileLoadBucket
func nativeProfileLoadBucket(int) uint64

//go:linkname nativeProfileShardsBaseAddr github.com/xgo-dev/llgo/runtime/internal/runtime.memProfileShardsBaseAddr
func nativeProfileShardsBaseAddr() uintptr

const nativeProfileTestBucket = 26 // 1 GiB bucket; unaffected by ordinary test allocations.

func TestNativeProfileShardBaseAlignment(t *testing.T) {
	base := nativeProfileShardsBaseAddr()
	if base == 0 || base%256 != 0 {
		t.Fatalf("memProfileShardsData base address %#x is not aligned to 256-byte shard stride / 128-byte cache lines", base)
	}
}

func TestNativeProfileSizeClasses(t *testing.T) {
	expected := func(size uintptr) (int, uintptr) {
		idx := 0
		class := uintptr(16)
		for class < size && class < 1<<30 {
			class *= 2
			idx++
		}
		return idx, class
	}
	for size := uintptr(0); size < 131072; size++ {
		wantIdx, wantClass := expected(size)
		if got := nativeProfileBucketIndex(size); got != wantIdx {
			t.Fatalf("size %d: bucket %d, expected %d", size, got, wantIdx)
		}
		if got := nativeProfileSizeClass(size); got != wantClass {
			t.Fatalf("size %d: class %d, expected %d", size, got, wantClass)
		}
	}
	for bit := uint(4); bit < uint(unsafe.Sizeof(uintptr(0))*8); bit++ {
		boundary := uintptr(1) << bit
		for _, size := range []uintptr{boundary - 1, boundary, boundary + 1} {
			wantIdx, wantClass := expected(size)
			if got := nativeProfileBucketIndex(size); got != wantIdx {
				t.Fatalf("boundary %d: bucket %d, expected %d", size, got, wantIdx)
			}
			if got := nativeProfileSizeClass(size); got != wantClass {
				t.Fatalf("boundary %d: class %d, expected %d", size, got, wantClass)
			}
		}
	}
	if got := nativeProfileBucketIndex(^uintptr(0)); got != nativeProfileTestBucket {
		t.Fatal("largest bucket", got)
	}
	if got := nativeProfileSizeClass(^uintptr(0)); got != 1<<30 {
		t.Fatal("largest class", got)
	}
}

func TestNativeProfileShardSpread(t *testing.T) {
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
			shard := nativeProfileShardIndex(base + i*stride.step)
			if shard >= 64 {
				t.Fatalf("%s: shard %d out of range", stride.name, shard)
			}
			seen[shard] = true
		}
		if len(seen) < stride.minUn {
			t.Fatalf("%s: only %d unique shards across 32 threads, want >= %d", stride.name, len(seen), stride.minUn)
		}
	}
}

func TestNativeProfileConcurrentCounterSnapshots(t *testing.T) {
	base := nativeProfileLoadBucket(nativeProfileTestBucket)
	const workers, iterations = 8, 20000
	for round := uint64(0); round < 3; round++ {
		var done sync.WaitGroup
		done.Add(workers)
		for range workers {
			go func() {
				defer done.Done()
				for range iterations {
					nativeProfileAddBucket(nativeProfileTestBucket)
				}
			}()
		}
		joined := make(chan struct{})
		go func() { done.Wait(); close(joined) }()
		previous := base + round*workers*iterations
		limit := base + (round+1)*workers*iterations
	polling:
		for {
			current := nativeProfileLoadBucket(nativeProfileTestBucket)
			if current < previous || current > limit {
				t.Fatal("nonmonotonic snapshot", previous, current, limit)
			}
			previous = current
			select {
			case <-joined:
				break polling
			default:
				runtime.Gosched()
			}
		}
		if got := nativeProfileLoadBucket(nativeProfileTestBucket); got != limit {
			t.Fatal("lost events after worker exit", round, got, limit)
		}
	}
}

var nativeProfileRetained [16]*[262144]byte

func TestNativeProfilePublicAllocationRecords(t *testing.T) {
	t.Cleanup(func() { clear(nativeProfileRetained[:]) })
	var records [64]runtime.MemProfileRecord
	objects := func() int64 {
		n, ok := runtime.MemProfile(records[:], true)
		if !ok {
			t.Fatal("profile capacity")
		}
		for _, record := range records[:n] {
			if record.AllocObjects > 0 && record.AllocBytes/record.AllocObjects == 262144 {
				return record.AllocObjects
			}
		}
		return 0
	}
	before := objects()
	for i := range nativeProfileRetained {
		p := new([262144]byte)
		p[0] = byte(i)
		nativeProfileRetained[i] = p
	}
	after := objects()
	if after-before < 16 {
		t.Fatal("public profile lost managed allocations", before, after)
	}
	runtime.KeepAlive(nativeProfileRetained)
}

func BenchmarkNativeProfileCounter(b *testing.B) {
	for _, workers := range []int{1, 4} {
		name := "serial"
		if workers == 4 {
			name = "four-workers"
		}
		b.Run(name, func(b *testing.B) {
			before := nativeProfileLoadBucket(nativeProfileTestBucket)
			var ready, done sync.WaitGroup
			ready.Add(workers)
			done.Add(workers)
			start := make(chan struct{})
			for worker := range workers {
				count := b.N / workers
				if worker < b.N%workers {
					count++
				}
				go func() {
					defer done.Done()
					ready.Done()
					<-start
					for range count {
						nativeProfileAddBucket(nativeProfileTestBucket)
					}
				}()
			}
			ready.Wait()
			b.ResetTimer()
			close(start)
			done.Wait()
			b.StopTimer()
			if got := nativeProfileLoadBucket(nativeProfileTestBucket) - before; got != uint64(b.N) {
				b.Fatal("counter events", got, b.N)
			}
		})
	}
}
