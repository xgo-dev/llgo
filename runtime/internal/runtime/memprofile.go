package runtime

import (
	"unsafe"
)

// MemProfileRecord describes allocations aggregated into one profile bucket.
type MemProfileRecord struct {
	AllocBytes, FreeBytes     int64
	AllocObjects, FreeObjects int64
	Stack0                    [32]uintptr
}

func (r *MemProfileRecord) InUseBytes() int64 {
	return r.AllocBytes - r.FreeBytes
}

func (r *MemProfileRecord) InUseObjects() int64 {
	return r.AllocObjects - r.FreeObjects
}

func (r *MemProfileRecord) Stack() []uintptr {
	for i, pc := range r.Stack0 {
		if pc == 0 {
			return r.Stack0[:i]
		}
	}
	return r.Stack0[:]
}

type memProfileBucket struct {
	size uintptr

	objects memProfileCounter
}

var memProfileBuckets = [...]memProfileBucket{
	{size: 16},
	{size: 32},
	{size: 64},
	{size: 128},
	{size: 256},
	{size: 512},
	{size: 1024},
	{size: 2048},
	{size: 4096},
	{size: 8192},
	{size: 16384},
	{size: 32768},
	{size: 65536},
	{size: 131072},
	{size: 262144},
	{size: 524288},
	{size: 1048576},
	{size: 2097152},
	{size: 4194304},
	{size: 8388608},
	{size: 16777216},
	{size: 33554432},
	{size: 67108864},
	{size: 134217728},
	{size: 268435456},
	{size: 536870912},
	{size: 1073741824},
}

// MemProfileStackCapture, set by the public runtime package, walks the
// physical stack at a sampled allocation. When present, MemProfile reports
// stack-keyed buckets (gc semantics: heapsampling.go attributes sampled
// bytes to call stacks); without it the legacy size-class counters remain
// (baremetal, wasm).
var MemProfileStackCapture func(pcs []uintptr) int

// MemProfileRatePtr points at the public runtime.MemProfileRate variable
// (user-settable at any time, read per sampling decision).
var MemProfileRatePtr *int

type memStackBucket struct {
	next         *memStackBucket
	hash         uintptr
	size         uintptr
	allocBytes   memProfileCounter
	allocObjects memProfileCounter
	nstk         int32
	stk          [32]uintptr
}

type memProfileThreadState struct {
	remaining uintptr
	rand      uint64
}

const memStackTabSize = 512 // power of two

var (
	memStackTab     [memStackTabSize]memProfileBucketHead
	memStackTabLock memProfileLock
)

func sampleMemProfileStack(size uintptr) {
	// Tiny allocations occupy at least one 16-byte granule (bdwgc's
	// minimum on 64-bit, matching gc's tiny size class): report the
	// granule so bytes/objects ratios match what the allocator really
	// spends — pprof consumers and the tiny-allocation tests key on it.
	if size < 16 {
		size = 16
	}
	var pcs [32]uintptr
	n := MemProfileStackCapture(pcs[:])
	if n <= 0 {
		return
	}
	var h uintptr = 5381
	for i := 0; i < n; i++ {
		h = h*33 + pcs[i]
	}
	// gc keys heap profile buckets by both stack and allocation size. Besides
	// preserving that contract, keeping sizes separate lets the consumer's
	// Poisson correction use the actual sample size instead of a mixed mean.
	h = h*33 + size
	slot := h & (memStackTabSize - 1)
	if b := findMemStackBucket(memProfileLoadBucket(&memStackTab[slot]), h, size, pcs[:n]); b != nil {
		memStackAdd(b, size)
		return
	}

	// Allocation re-enters recordMemProfileAlloc, where this physical
	// thread's recursion guard suppresses the internal allocation. Keep the
	// allocation outside the table lock so allocator or finalizer work cannot
	// invert lock order with a concurrent MemProfile call.
	b := (*memStackBucket)(AllocZ(unsafe.Sizeof(memStackBucket{})))
	b.hash = h
	b.size = size
	b.nstk = int32(n)
	copy(b.stk[:], pcs[:n])

	memStackTabLock.lock()
	head := memProfileLoadBucket(&memStackTab[slot])
	if existing := findMemStackBucket(head, h, size, pcs[:n]); existing != nil {
		memStackTabLock.unlock()
		memStackAdd(existing, size)
		return
	}
	memStackAdd(b, size)
	b.next = head
	memProfileStoreBucket(&memStackTab[slot], b)
	memStackTabLock.unlock()
}

// Buckets and their next links are immutable after the head is atomically
// published, so lookups need no lock. The insertion lock only resolves the
// rare race between two first samples for the same key.
func findMemStackBucket(head *memStackBucket, hash, size uintptr, pcs []uintptr) *memStackBucket {
	for b := head; b != nil; b = b.next {
		if b.hash == hash && b.size == size && int(b.nstk) == len(pcs) && memStackEqual(b, pcs) {
			return b
		}
	}
	return nil
}

func memStackEqual(b *memStackBucket, pcs []uintptr) bool {
	for i := range pcs {
		if b.stk[i] != pcs[i] {
			return false
		}
	}
	return true
}

// memStackAdd records one raw sampled allocation (gc semantics: no
// scaling here; readers un-bias with the sampling-rate correction).
func memStackAdd(b *memStackBucket, size uintptr) {
	memProfileAddN(&b.allocObjects, 1)
	memProfileAddN(&b.allocBytes, uint64(size))
}

func memProfileSizeClass(size uintptr) uintptr {
	if size <= 16 {
		return 16
	}
	for _, b := range memProfileBuckets {
		if size <= b.size {
			return b.size
		}
	}
	return memProfileBuckets[len(memProfileBuckets)-1].size
}

func MemProfile(p []MemProfileRecord, inuseZero bool) (n int, ok bool) {
	if MemProfileStackCapture != nil && MemProfileRatePtr != nil {
		return memProfileStacks(p)
	}
	for i := range memProfileBuckets {
		if memProfileLoadObjects(&memProfileBuckets[i].objects) != 0 {
			n++
		}
	}
	if len(p) < n {
		return n, false
	}
	j := 0
	for i := range memProfileBuckets {
		b := &memProfileBuckets[i]
		objects := memProfileLoadObjects(&b.objects)
		if objects == 0 {
			continue
		}
		p[j] = MemProfileRecord{
			AllocBytes:   int64(uint64(b.size) * uint64(objects)),
			AllocObjects: int64(objects),
		}
		j++
	}
	return n, true
}

func memProfileStacks(p []MemProfileRecord) (n int, ok bool) {
	// Save one atomic head per slot. Published chains are immutable, so this
	// gives a stable snapshot without blocking allocation samples and without
	// touching the caller thread's sampling state.
	var heads [memStackTabSize]*memStackBucket
	for i := range heads {
		heads[i] = memProfileLoadBucket(&memStackTab[i])
		for b := heads[i]; b != nil; b = b.next {
			n++
		}
	}
	if len(p) < n {
		return n, false
	}
	j := 0
	for i := range heads {
		for b := heads[i]; b != nil; b = b.next {
			r := MemProfileRecord{
				AllocBytes:   int64(memProfileLoadObjects(&b.allocBytes)),
				AllocObjects: int64(memProfileLoadObjects(&b.allocObjects)),
			}
			copy(r.Stack0[:], b.stk[:b.nstk])
			p[j] = r
			j++
		}
	}
	return n, true
}
