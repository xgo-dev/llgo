//go:build baremetal

package runtime

// Baremetal accounting uses a non-atomic native-word counter.
var memProfileCounters [memProfileNumBuckets]uintptr

func memProfileAddBucket(idx int) {
	memProfileCounters[idx]++
}

func memProfileLoadBucket(idx int) uint64 {
	return uint64(memProfileCounters[idx])
}
