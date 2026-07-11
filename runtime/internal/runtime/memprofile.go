package runtime

import "unsafe"

// MemProfileRecord describes allocations aggregated by size class.
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
	allocBytes   memProfileCounter
	allocObjects memProfileCounter
	nstk         int32
	stk          [32]uintptr
}

const memStackTabSize = 512 // power of two

var memStackTab [memStackTabSize]*memStackBucket

// memProfileRemaining counts down allocated bytes to the next sample.
// Thresholds are drawn uniformly from [1, 2*rate) — mean rate — because a
// deterministic stride starves small allocation sites when they interleave
// with larger ones (every crossing lands on the big sites; gc randomizes
// for the same reason). Benign races: sampling is statistical.
var (
	memProfileRemaining uintptr
	memProfileRandState uint64 = 0x9e3779b97f4a7c15
)

func memProfileNextThreshold(rate int) uintptr {
	// Exponentially distributed with mean rate, like gc's fastexprand:
	// the memoryless property is required — with any bounded-support
	// distribution a near-periodic allocation pattern phase-locks the
	// sampling points onto the large sites and skews per-site estimates
	// (observed 1.6x on goroot heapsampling's interleaved sizes).
	x := memProfileRandState
	x ^= x >> 12
	x ^= x << 25
	x ^= x >> 27
	memProfileRandState = x
	r := (x * 0x2545f4914f6cdd1d) >> 11 // 53 random bits
	u := float64(r) / (1 << 53)
	if u < 1e-12 {
		u = 1e-12
	}
	t := -lnApprox(u) * float64(rate)
	if t < 1 {
		t = 1
	}
	if max := float64(rate) * 64; t > max {
		t = max
	}
	return uintptr(t)
}

// lnApprox computes ln(u) for u in (0,1] via exponent split and an
// atanh series on the mantissa — a few 1e-6s of relative error, far
// below sampling noise.
func lnApprox(u float64) float64 {
	const ln2 = 0.6931471805599453
	bits := *(*uint64)(unsafe.Pointer(&u))
	e := int((bits>>52)&0x7ff) - 1023
	mbits := (bits &^ (uint64(0x7ff) << 52)) | (uint64(1023) << 52)
	m := *(*float64)(unsafe.Pointer(&mbits)) // in [1, 2)
	z := (m - 1) / (m + 1)
	z2 := z * z
	lnm := 2 * z * (1 + z2/3 + z2*z2/5 + z2*z2*z2/7)
	return float64(e)*ln2 + lnm
}

// memProfileInSample breaks the recursion: allocating a bucket node (and
// anything the capture path allocates) re-enters recordMemProfileAlloc.
// Benign-racy flag — a concurrent thread skipping one sample is fine.
var memProfileInSample bool

func recordMemProfileAlloc(size uintptr) {
	if size == 0 {
		return
	}
	if MemProfileStackCapture != nil && MemProfileRatePtr != nil {
		// The guard covers the whole decision path: threshold drawing and
		// stack capture may themselves allocate (escaping locals, the
		// bucket node), and a recursive sample would overflow the stack.
		if memProfileInSample {
			return
		}
		memProfileInSample = true
		rate := *MemProfileRatePtr
		if rate <= 0 {
			memProfileInSample = false
			return
		}
		if rate == 1 {
			sampleMemProfileStack(size)
			memProfileInSample = false
			return
		}
		// Mirror gc's mcache.nextSample: subtract, sample once on
		// crossing, redraw. Records hold RAW sampled counts — consumers
		// (pprof, goroot heapsampling.go) apply the Poisson correction
		// (scaleHeapSample) themselves, exactly like with gc.
		if memProfileRemaining == 0 {
			memProfileRemaining = memProfileNextThreshold(rate)
		}
		if size < memProfileRemaining {
			memProfileRemaining -= size
			memProfileInSample = false
			return
		}
		memProfileRemaining = memProfileNextThreshold(rate)
		sampleMemProfileStack(size)
		memProfileInSample = false
		return
	}
	sizeClass := memProfileSizeClass(size)
	for i := range memProfileBuckets {
		b := &memProfileBuckets[i]
		if b.size == sizeClass {
			memProfileAddObject(&b.objects)
			return
		}
	}
}

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
	slot := h & (memStackTabSize - 1)
	for b := memStackTab[slot]; b != nil; b = b.next {
		if b.hash == h && int(b.nstk) == n && memStackEqual(b, pcs[:n]) {
			memStackAdd(b, size)
			return
		}
	}
	b := (*memStackBucket)(AllocZ(unsafe.Sizeof(memStackBucket{})))
	b.hash = h
	b.nstk = int32(n)
	copy(b.stk[:], pcs[:n])
	memStackAdd(b, size)
	// Benign-racy publish: a lost insert loses one sample, never corrupts
	// (nodes are immutable once linked and the list is prepend-only).
	b.next = memStackTab[slot]
	memStackTab[slot] = b
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
	// Freeze sampling while enumerating so the count cannot grow between
	// a caller's sizing call and its fill call. Explicit reset instead of
	// defer: the wasm backend crashes in instruction selection on the
	// deferred closure here.
	memProfileInSample = true
	n, ok = memProfileStacksLocked(p)
	memProfileInSample = false
	return n, ok
}

func memProfileStacksLocked(p []MemProfileRecord) (n int, ok bool) {
	for i := range memStackTab {
		for b := memStackTab[i]; b != nil; b = b.next {
			n++
		}
	}
	if len(p) < n {
		return n, false
	}
	j := 0
	for i := range memStackTab {
		for b := memStackTab[i]; b != nil; b = b.next {
			if j >= len(p) {
				break
			}
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
