//go:build llgo && !baremetal && !nogc

package runtime_test

import (
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"
)

func heapRecords(t *testing.T) []runtime.MemProfileRecord {
	t.Helper()
	n, _ := runtime.MemProfile(nil, true)
	for tries := 0; tries < 4; tries++ {
		records := make([]runtime.MemProfileRecord, n+16)
		if count, ok := runtime.MemProfile(records, true); ok {
			return records[:count]
		} else {
			n = count
		}
	}
	t.Fatal("profile kept growing during snapshot")
	return nil
}

func heapRecordHasFunction(record runtime.MemProfileRecord, name string) bool {
	frames := runtime.CallersFrames(record.Stack())
	for {
		frame, more := frames.Next()
		if strings.HasSuffix(frame.Function, "."+name) {
			return true
		}
		if !more {
			return false
		}
	}
}

var sampledBuffers [80][]byte

//go:noinline
func allocateProfileSize(size int) []byte { return make([]byte, size) }

func TestHeapProfileSamplingRateAndManyRecords(t *testing.T) {
	old := runtime.MemProfileRate
	defer func() { runtime.MemProfileRate = old; sampledBuffers = [80][]byte{} }()
	runtime.MemProfileRate = 0
	before := len(heapRecords(t))
	for i := range sampledBuffers {
		sampledBuffers[i] = allocateProfileSize(128 + i*32)
	}
	if after := len(heapRecords(t)); after != before {
		t.Fatalf("disabled sampling added %d records", after-before)
	}
	runtime.MemProfileRate = 1
	for i := range sampledBuffers {
		sampledBuffers[i] = allocateProfileSize(128 + i*32)
	}
	runtime.MemProfileRate = 0
	records := heapRecords(t)
	sizes := make(map[int64]bool)
	for _, r := range records {
		if heapRecordHasFunction(r, "allocateProfileSize") && r.AllocObjects != 0 {
			frame, _ := runtime.CallersFrames(r.Stack()).Next()
			if !strings.HasSuffix(frame.Function, ".allocateProfileSize") {
				t.Fatalf("profile starts in allocator plumbing: %+v", frame)
			}
			sizes[r.AllocBytes/r.AllocObjects] = true
		}
	}
	if len(sizes) != len(sampledBuffers) {
		t.Fatalf("rate=1 recorded %d distinct sizes, want %d", len(sizes), len(sampledBuffers))
	}
	if n, ok := runtime.MemProfile(make([]runtime.MemProfileRecord, 1), true); ok || n <= 64 {
		t.Fatalf("short profile buffer: n=%d ok=%v", n, ok)
	}
	runtime.KeepAlive(sampledBuffers)
}

//go:linkname allocateProfileManagedBytes github.com/xgo-dev/llgo/runtime/internal/runtime.AllocAtomicU
func allocateProfileManagedBytes(uintptr) unsafe.Pointer

var transientProfileChecksum byte

//go:noinline
func allocateTransientProfile() {
	// Force a real managed allocation independently of Go escape elision.
	p := allocateProfileManagedBytes(65536)
	*(*byte)(p) = 17
	transientProfileChecksum = *(*byte)(p)
	runtime.KeepAlive(p)
}

func TestHeapProfileAccountsForReclamation(t *testing.T) {
	old := runtime.MemProfileRate
	defer func() { runtime.MemProfileRate = old }()
	runtime.MemProfileRate = 0
	var beforeFree int64
	for _, record := range heapRecords(t) {
		if heapRecordHasFunction(record, "allocateTransientProfile") {
			beforeFree += record.FreeObjects
		}
	}
	runtime.MemProfileRate = 1
	done := make(chan struct{})
	go func() {
		// Conservative collectors may retain an individual object because a
		// stale word resembles its address. Observe reclamation of a batch
		// instead of requiring a precise lifetime for one chosen object.
		for i := 0; i < 8; i++ {
			allocateTransientProfile()
		}
		close(done)
	}()
	<-done
	if transientProfileChecksum != 17 {
		t.Fatal("allocation was not used")
	}
	runtime.MemProfileRate = 0
	for tries := 0; tries < 20; tries++ {
		runtime.GC()
		var freed int64
		for _, r := range heapRecords(t) {
			if heapRecordHasFunction(r, "allocateTransientProfile") {
				freed += r.FreeObjects
			}
		}
		if freed > beforeFree {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("no sampled allocations were reclaimed after their goroutine exited")
}

//go:noinline
func allocateConcurrentProfile(size int) []byte { return make([]byte, size) }

func TestHeapProfileConcurrentAllocations(t *testing.T) {
	old := runtime.MemProfileRate
	defer func() { runtime.MemProfileRate = old }()
	runtime.MemProfileRate = 0
	var before int64
	for _, r := range heapRecords(t) {
		if heapRecordHasFunction(r, "allocateConcurrentProfile") {
			before += r.AllocObjects
		}
	}
	runtime.MemProfileRate = 1
	const workers, count = 4, 40
	var retained [workers][count][]byte
	var done sync.WaitGroup
	done.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func(worker int) {
			defer done.Done()
			for i := 0; i < count; i++ {
				retained[worker][i] = allocateConcurrentProfile(4096)
			}
		}(worker)
	}
	done.Wait()
	runtime.MemProfileRate = 0
	var objects int64
	for _, r := range heapRecords(t) {
		if heapRecordHasFunction(r, "allocateConcurrentProfile") {
			objects += r.AllocObjects
		}
	}
	if objects-before != workers*count {
		t.Fatalf("concurrent samples=%d want=%d", objects-before, workers*count)
	}
	runtime.KeepAlive(retained)
}
