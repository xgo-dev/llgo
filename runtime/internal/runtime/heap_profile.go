//go:build !nogc && !baremetal

package runtime

import (
	"unsafe"

	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

// Buckets persist, as in Go's profiler. Their numeric PC arrays live in
// unscanned storage so synthetic Wasm PCs cannot become conservative roots.
type heapProfileBucket struct {
	next                      *heapProfileBucket
	size                      uintptr
	stack                     *[32]uintptr
	allocBytes, freeBytes     int64
	allocObjects, freeObjects int64
}

type heapProfileSample struct {
	key    uintptr // encoded object identity; the collector clears this word
	next   *heapProfileSample
	bucket *heapProfileBucket
	weak   heapProfileWeakLink
}

var heapProfiles struct {
	once   psync.Once
	mu     psync.Mutex
	table  [256]*heapProfileBucket
	head   *heapProfileSample
	tail   *heapProfileSample
	free   *heapProfileSample
	active int
}

func initHeapProfiles() { heapProfiles.mu.Init(nil) }

// PauseHeapProfile suppresses only this G's profiler bookkeeping. Exact
// allocation counters exclude this bookkeeping; other Gs continue sampling.
func PauseHeapProfile() unsafe.Pointer {
	gp := heapProfileG()
	if gp != nil {
		gp.heapProfile.depth++
	}
	return unsafe.Pointer(gp)
}

func ResumeHeapProfile(token unsafe.Pointer) {
	if gp := (*g)(token); gp != nil {
		gp.heapProfile.depth--
	}
}

func recordHeapProfileAlloc(gp *g, ptr unsafe.Pointer, size uintptr) {
	rate := MemProfileRate
	if rate <= 0 {
		return
	}
	size = heapProfileObjectSize(ptr, size)
	s := &gp.heapProfile
	// Keep initialization and random-number generation off ordinary allocations.
	if s.rate == rate && uint64(size) < s.next {
		s.next -= uint64(size)
		return
	}
	if !heapProfileShouldSample(s, rate, size, uint64(uintptr(unsafe.Pointer(gp)))^gp.goid) {
		return
	}
	recordHeapProfileSample(s, ptr, size)
}

// Keep defer registration off the unsampled allocation and G-bootstrap paths:
// registering a defer itself needs the current G.
//
//go:noinline
func recordHeapProfileSample(s *heapProfileGState, ptr unsafe.Pointer, size uintptr) {
	s.depth++
	defer func() { s.depth-- }()
	if s.pcs == nil {
		s.pcs = (*[32]uintptr)(allocHeapProfilePCs(unsafe.Sizeof(*s.pcs)))
	}
	n := captureHeapProfilePCs(s.pcs[:])
	if n == 0 {
		return
	}
	pinHeapProfilePCs(s.pcs[:n])
	for i := n; i < len(s.pcs); i++ {
		s.pcs[i] = 0
	}
	heapProfiles.once.Do(initHeapProfiles)
	heapProfiles.mu.Lock()
	// Rotate through a bounded batch instead of rescanning all live samples
	// on every allocation. A snapshot below drains the complete queue.
	drainHeapProfileSamples(16)
	bucket := findHeapProfileBucket(size, s.pcs, n)
	sample := heapProfiles.free
	if sample != nil {
		heapProfiles.free = sample.next
	} else {
		sample = (*heapProfileSample)(AllocRoot(unsafe.Sizeof(heapProfileSample{})))
	}
	sample.next = nil
	sample.bucket = bucket
	sample.key = ^uintptr(ptr)
	if !registerHeapProfileWeak(sample, ptr) {
		sample.bucket = nil
		sample.next = heapProfiles.free
		heapProfiles.free = sample
		heapProfiles.mu.Unlock()
		return
	}
	bucket.allocObjects++
	bucket.allocBytes += int64(size)
	appendHeapProfileSample(sample)
	heapProfiles.active++
	heapProfiles.mu.Unlock()
}

func appendHeapProfileSample(s *heapProfileSample) {
	s.next = nil
	if heapProfiles.tail == nil {
		heapProfiles.head = s
	} else {
		heapProfiles.tail.next = s
	}
	heapProfiles.tail = s
}

func drainHeapProfileSamples(limit int) {
	if limit > heapProfiles.active {
		limit = heapProfiles.active
	}
	for i := 0; i < limit; i++ {
		s := heapProfiles.head
		heapProfiles.head = s.next
		if heapProfiles.head == nil {
			heapProfiles.tail = nil
		}
		if atomic.Load(&s.key) == 0 {
			s.bucket.freeObjects++
			s.bucket.freeBytes += int64(s.bucket.size)
			s.bucket = nil
			s.next = heapProfiles.free
			heapProfiles.free = s
			heapProfiles.active--
		} else {
			appendHeapProfileSample(s)
		}
	}
}

func findHeapProfileBucket(size uintptr, pcs *[32]uintptr, n int) *heapProfileBucket {
	hash := uint64(size) ^ 0xcbf29ce484222325
	for i := 0; i < n; i++ {
		hash = (hash ^ uint64(pcs[i])) * 0x100000001b3
	}
	index := hash % uint64(len(heapProfiles.table))
	for b := heapProfiles.table[index]; b != nil; b = b.next {
		if b.size == size && *b.stack == *pcs {
			return b
		}
	}
	b := (*heapProfileBucket)(AllocRoot(unsafe.Sizeof(heapProfileBucket{})))
	b.stack = (*[32]uintptr)(allocHeapProfilePCs(unsafe.Sizeof(*pcs)))
	*b.stack = *pcs
	b.size = size
	b.next = heapProfiles.table[index]
	heapProfiles.table[index] = b
	return b
}

func readHeapProfile(p []MemProfileRecord, inuseZero bool) (n int, ok bool) {
	token := PauseHeapProfile()
	defer ResumeHeapProfile(token)
	heapProfiles.once.Do(initHeapProfiles)
	heapProfiles.mu.Lock()
	defer heapProfiles.mu.Unlock()
	drainHeapProfileSamples(heapProfiles.active)
	for _, head := range heapProfiles.table {
		for b := head; b != nil; b = b.next {
			if b.allocObjects != 0 && (inuseZero || b.allocObjects != b.freeObjects) {
				n++
			}
		}
	}
	if len(p) < n {
		return n, false
	}
	i := 0
	for _, head := range heapProfiles.table {
		for b := head; b != nil; b = b.next {
			if b.allocObjects == 0 || !inuseZero && b.allocObjects == b.freeObjects {
				continue
			}
			p[i] = MemProfileRecord{
				AllocBytes: b.allocBytes, FreeBytes: b.freeBytes,
				AllocObjects: b.allocObjects, FreeObjects: b.freeObjects,
				Stack0: *b.stack,
			}
			i++
		}
	}
	return n, true
}

func heapProfileShouldSample(s *heapProfileGState, rate int, size uintptr, seed uint64) bool {
	if s.rand == 0 {
		s.rand = seed ^ 0x9e3779b97f4a7c15
		if s.rand == 0 {
			s.rand = 1
		}
	}
	if s.rate != rate {
		s.rate = rate
		s.next = nextHeapProfileSample(s, rate)
	}
	if uint64(size) < s.next {
		s.next -= uint64(size)
		return false
	}
	s.next = nextHeapProfileSample(s, rate)
	return true
}

// This is Go runtime's exponential sampling algorithm (malloc.go), using
// its fastlog2 approximation and a per-G xorshift generator.
func nextHeapProfileSample(s *heapProfileGState, mean int) uint64 {
	if mean == 1 {
		return 0
	}
	if mean > 0x7000000 {
		mean = 0x7000000
	}
	x := s.rand
	x ^= x >> 12
	x ^= x << 25
	x ^= x >> 27
	s.rand = x
	q := uint32((x*0x2545f4914f6cdd1d)>>38) + 1
	log := heapProfileLog2(float64(q)) - 26
	if log > 0 {
		log = 0
	}
	return uint64(log*(-0.6931471805599453*float64(mean))) + 1
}

func heapProfileFloat64bits(f float64) uint64 { return *(*uint64)(unsafe.Pointer(&f)) }

func releaseHeapProfileG(gp *g) {
	if gp.heapProfile.pcs != nil {
		freeHeapProfilePCs(unsafe.Pointer(gp.heapProfile.pcs))
		gp.heapProfile.pcs = nil
	}
}
