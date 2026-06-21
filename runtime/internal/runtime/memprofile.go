package runtime

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

func recordMemProfileAlloc(size uintptr) {
	if size == 0 {
		return
	}
	size = memProfileSizeClass(size)
	for i := range memProfileBuckets {
		b := &memProfileBuckets[i]
		if b.size == size {
			memProfileAddObject(&b.objects)
			return
		}
	}
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
