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

type memProfileBucket struct {
	size    uintptr
	objects memProfileCounter
}

const memProfileMaxSizeClass = uintptr(1) << 30

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
	{size: memProfileMaxSizeClass},
}

// Centralize class selection and counter updates across allocator entry points.
//
//go:noinline
func recordMemProfileAlloc(size uintptr) {
	if size != 0 {
		memProfileAddObject(&memProfileBuckets[memProfileBucketIndex(size)].objects)
	}
}

func memProfileBucketIndex(size uintptr) int {
	if size <= 16 {
		return 0
	}
	if size > memProfileMaxSizeClass {
		return len(memProfileBuckets) - 1
	}
	x := uint64(size-1) >> 4
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

func memProfileSizeClass(size uintptr) uintptr {
	return memProfileBuckets[memProfileBucketIndex(size)].size
}

func MemProfile(p []MemProfileRecord, inuseZero bool) (n int, ok bool) {
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
			ObjectSize:   int64(b.size),
			AllocBytes:   int64(uint64(b.size) * uint64(objects)),
			AllocObjects: int64(objects),
		}
		j++
	}
	return n, true
}
