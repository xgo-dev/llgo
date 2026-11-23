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
	memProfileMinSizeShift = 4
	memProfileMaxSizeShift = 30
	memProfileMaxSizeClass = uintptr(1) << memProfileMaxSizeShift
)

// Bucket sizes are powers of two. Keep the counters in zero-initialized
// storage so their padding and shards need no bytes in the executable.
var memProfileBuckets [memProfileMaxSizeShift - memProfileMinSizeShift + 1]memProfileCounter

// Keep class selection and atomic updates out of allocator wrappers to limit
// duplicated code and native binary-size variation.
//
//go:noinline
func recordMemProfileAlloc(size uintptr) {
	if size != 0 {
		memProfileAddObject(&memProfileBuckets[memProfileBucketIndex(size)])
	}
}

func memProfileBucketIndex(size uintptr) int {
	if size <= 1<<memProfileMinSizeShift {
		return 0
	}
	if size > memProfileMaxSizeClass {
		return len(memProfileBuckets) - 1
	}
	x := uint64(size-1) >> memProfileMinSizeShift
	index := 1
	if x>>16 != 0 {
		x >>= 16
		index += 16
	}
	if x>>8 != 0 {
		x >>= 8
		index += 8
	}
	if x>>4 != 0 {
		x >>= 4
		index += 4
	}
	if x>>2 != 0 {
		x >>= 2
		index += 2
	}
	if x>>1 != 0 {
		index++
	}
	return index
}

// memProfileSizeClass exposes the bucket mapping to the native profile tests.
func memProfileSizeClass(size uintptr) uintptr {
	return uintptr(1) << (memProfileMinSizeShift + memProfileBucketIndex(size))
}

// MemProfileMallocs returns recorded managed-allocation events across all
// size classes. Reuse the allocator's existing counters without allocating
// a profile snapshot or adding work to allocation call sites.
func MemProfileMallocs() uint64 {
	var total uint64
	for i := range memProfileBuckets {
		total += memProfileLoadObjects(&memProfileBuckets[i])
	}
	return total
}

func MemProfile(p []MemProfileRecord, inuseZero bool) (n int, ok bool) {
	for i := range memProfileBuckets {
		if memProfileLoadObjects(&memProfileBuckets[i]) != 0 {
			n++
		}
	}
	if len(p) < n {
		return n, false
	}
	j := 0
	for i := range memProfileBuckets {
		objects := memProfileLoadObjects(&memProfileBuckets[i])
		if objects == 0 {
			continue
		}
		size := uint64(1) << (memProfileMinSizeShift + i)
		p[j] = MemProfileRecord{
			ObjectSize:   int64(size),
			AllocBytes:   int64(size * objects),
			AllocObjects: int64(objects),
		}
		j++
	}
	return n, true
}
