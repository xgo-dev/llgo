//go:build !llgo || baremetal || wasm

package runtime

func memProfileBucketIndex(size uintptr) int {
	if size <= 16 {
		return 0
	}
	if size >= memProfileMaxSizeClass {
		return memProfileNumBuckets - 1
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
