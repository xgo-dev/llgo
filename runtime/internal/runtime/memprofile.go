package runtime

// MemProfileRecord describes allocations aggregated by size class.
type MemProfileRecord struct {
	ObjectSize                int64
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

const (
	memProfileMaxSizeClass = uintptr(1) << 30
	memProfileNumBuckets   = 27
)

var memProfileBucketSizes = [memProfileNumBuckets]uintptr{
	16,
	32,
	64,
	128,
	256,
	512,
	1024,
	2048,
	4096,
	8192,
	16384,
	32768,
	65536,
	131072,
	262144,
	524288,
	1048576,
	2097152,
	4194304,
	8388608,
	16777216,
	33554432,
	67108864,
	134217728,
	268435456,
	536870912,
	memProfileMaxSizeClass,
}

// Keep class selection and atomic updates out of allocator wrappers to limit
// duplicated code and native binary-size variation.
//
//go:noinline
func recordMemProfileAlloc(size uintptr) {
	if size != 0 {
		memProfileAddBucket(memProfileBucketIndex(size))
	}
}

// memProfileSizeClass exposes the bucket mapping to the native profile tests.
func memProfileSizeClass(size uintptr) uintptr {
	return memProfileBucketSizes[memProfileBucketIndex(size)]
}

// MemProfileMallocs returns recorded managed-allocation events across all
// size classes. Reuse the allocator's existing counters without allocating
// a profile snapshot or adding work to allocation call sites.
func MemProfileMallocs() uint64 {
	var total uint64
	for i := range memProfileBucketSizes {
		total += memProfileLoadBucket(i)
	}
	return total
}

func MemProfile(p []MemProfileRecord, inuseZero bool) (n int, ok bool) {
	for i := range memProfileBucketSizes {
		if memProfileLoadBucket(i) != 0 {
			n++
		}
	}
	if len(p) < n {
		return n, false
	}
	j := 0
	for i, size := range memProfileBucketSizes {
		objects := memProfileLoadBucket(i)
		if objects == 0 {
			continue
		}
		p[j] = MemProfileRecord{
			ObjectSize:   int64(size),
			AllocBytes:   int64(uint64(size) * uint64(objects)),
			AllocObjects: int64(objects),
		}
		j++
	}
	return n, true
}
