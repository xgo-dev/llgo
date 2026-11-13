//go:build llgo && !baremetal && !wasm

package runtime_test

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
)

//go:linkname nativeProfileSizeClass github.com/xgo-dev/llgo/runtime/internal/runtime.memProfileSizeClass
func nativeProfileSizeClass(uintptr) uintptr

//go:linkname nativeProfileAdd github.com/xgo-dev/llgo/runtime/internal/runtime.memProfileAddObject
func nativeProfileAdd(unsafe.Pointer)

//go:linkname nativeProfileLoad github.com/xgo-dev/llgo/runtime/internal/runtime.memProfileLoadObjects
func nativeProfileLoad(unsafe.Pointer) uint64

// The opaque storage follows the private native counter's 32 cache-line shards.
// The alignment marker also covers targets whose ordinary uint64 alignment is 4.
type nativeProfileCounterStorage struct {
	_     [0]atomic.Uint64
	words [32][16]uint64
}

func TestNativeProfileSizeClasses(t *testing.T) {
	expected := func(size uintptr) uintptr {
		class := uintptr(16)
		for class < size && class < 1<<30 {
			class *= 2
		}
		return class
	}
	for size := uintptr(0); size < 131072; size++ {
		if got := nativeProfileSizeClass(size); got != expected(size) {
			t.Fatalf("size %d: class %d, expected %d", size, got, expected(size))
		}
	}
	for bit := uint(4); bit < uint(unsafe.Sizeof(uintptr(0))*8); bit++ {
		boundary := uintptr(1) << bit
		for _, size := range []uintptr{boundary - 1, boundary, boundary + 1} {
			if got := nativeProfileSizeClass(size); got != expected(size) {
				t.Fatalf("boundary %d: class %d, expected %d", size, got, expected(size))
			}
		}
	}
	if got := nativeProfileSizeClass(^uintptr(0)); got != 1<<30 {
		t.Fatal("largest class", got)
	}
}

func TestNativeProfileConcurrentCounterSnapshots(t *testing.T) {
	var guarded struct {
		before  [16]uint64
		counter nativeProfileCounterStorage
		after   [16]uint64
	}
	for i := range guarded.before {
		guarded.before[i] = 0x123456789abcdef0
		guarded.after[i] = 0xfedcba9876543210
	}
	ptr := unsafe.Pointer(&guarded.counter)
	if uintptr(ptr)%8 != 0 || unsafe.Sizeof(guarded.counter) != 4096 {
		t.Fatal("native counter alignment or size")
	}
	if got := nativeProfileLoad(ptr); got != 0 {
		t.Fatal("initial counter", got)
	}
	const workers, iterations = 8, 20000
	for round := uint64(0); round < 3; round++ {
		var done sync.WaitGroup
		done.Add(workers)
		for range workers {
			go func() {
				defer done.Done()
				for range iterations {
					nativeProfileAdd(ptr)
				}
			}()
		}
		joined := make(chan struct{})
		go func() { done.Wait(); close(joined) }()
		previous := round * workers * iterations
	polling:
		for {
			current := nativeProfileLoad(ptr)
			if current < previous || current > (round+1)*workers*iterations {
				t.Fatal("nonmonotonic snapshot", previous, current)
			}
			previous = current
			select {
			case <-joined:
				break polling
			default:
				runtime.Gosched()
			}
		}
		if got := nativeProfileLoad(ptr); got != (round+1)*workers*iterations {
			t.Fatal("lost events after worker exit", round, got)
		}
	}
	for i := range guarded.before {
		if guarded.before[i] != 0x123456789abcdef0 || guarded.after[i] != 0xfedcba9876543210 {
			t.Fatal("counter escaped opaque storage", i)
		}
	}
	runtime.KeepAlive(&guarded)
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
			var counter nativeProfileCounterStorage
			ptr := unsafe.Pointer(&counter)
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
						nativeProfileAdd(ptr)
					}
				}()
			}
			ready.Wait()
			b.ResetTimer()
			close(start)
			done.Wait()
			b.StopTimer()
			if got := nativeProfileLoad(ptr); got != uint64(b.N) {
				b.Fatal("counter events", got, b.N)
			}
			runtime.KeepAlive(&counter)
		})
	}
}
