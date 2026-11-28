// Package pclntab contains the Go-style findfunc bucket/index algorithm used
// by LLGo runtime metadata. It is intentionally free of LLVM dependencies so
// build-time emitters and tests share one implementation of the pclntab logic.
package pclntab

import "fmt"

const (
	// These constants intentionally match Go's pclntab findfunc layout:
	// cmd/link builds one 4096-byte text bucket, split into 16 256-byte
	// subbuckets, and runtime.findfunc starts scanning from the recorded
	// bucket base plus subbucket delta.
	MinFuncSize       = uint32(16)
	FuncTabBucketSize = uint32(256) * MinFuncSize
	FindFuncSubbucket = 16
)

// FuncTabEntry mirrors the two pieces of data Go's linker stores in functab:
// a PC offset sorted by final text address, and an opaque function metadata
// offset. LLGo's current caller uses FuncOff as a payload index.
type FuncTabEntry struct {
	EntryOff uint32
	FuncOff  uint32
}

// FindFuncBucket mirrors runtime.findfuncbucket: one uint32 base function
// index plus 16 one-byte deltas into the sorted functab.
type FindFuncBucket struct {
	Idx        uint32
	Subbuckets [FindFuncSubbucket]uint8
}

// BuildFindFuncBuckets ports Go's cmd/link findfunctab construction for a
// sorted functab. It deliberately stays independent of LLVM so build/link code
// can use it without duplicating the algorithm.
func BuildFindFuncBuckets(ftab []FuncTabEntry, textSize uint32) ([]FindFuncBucket, error) {
	if textSize == 0 {
		return nil, nil
	}
	if len(ftab) < 2 {
		return nil, fmt.Errorf("pclntab ftab needs at least one function and one sentinel")
	}
	for i := 1; i < len(ftab); i++ {
		if ftab[i].EntryOff <= ftab[i-1].EntryOff {
			return nil, fmt.Errorf("pclntab ftab entries must be strictly increasing")
		}
	}
	if ftab[0].EntryOff != 0 {
		return nil, fmt.Errorf("pclntab first entry offset must be zero")
	}
	if ftab[len(ftab)-1].EntryOff < textSize {
		return nil, fmt.Errorf("pclntab sentinel offset %d below text size %d", ftab[len(ftab)-1].EntryOff, textSize)
	}

	nbuckets := int((textSize + FuncTabBucketSize - 1) / FuncTabBucketSize)
	buckets := make([]FindFuncBucket, nbuckets)
	subSize := FuncTabBucketSize / FindFuncSubbucket
	for b := range buckets {
		bucketStart := uint32(b) * FuncTabBucketSize
		baseIdx := FuncIndexForPC(ftab, bucketStart)
		buckets[b].Idx = uint32(baseIdx)
		for s := 0; s < FindFuncSubbucket; s++ {
			pc := bucketStart + uint32(s)*subSize
			if pc >= textSize {
				pc = textSize - 1
			}
			subIdx := FuncIndexForPC(ftab, pc)
			delta := subIdx - baseIdx
			if delta < 0 || delta > 255 {
				return nil, fmt.Errorf("pclntab subbucket delta overflow: bucket=%d subbucket=%d delta=%d", b, s, delta)
			}
			buckets[b].Subbuckets[s] = uint8(delta)
		}
	}
	return buckets, nil
}

// FuncIndexForPC is the slow reference lookup over the sorted functab. It is
// kept for tests and for building the compact bucket table.
func FuncIndexForPC(ftab []FuncTabEntry, pcOff uint32) int {
	lo, hi := 0, len(ftab)-1 // last entry is the sentinel.
	for lo+1 < hi {
		mid := int(uint(lo+hi) >> 1)
		if ftab[mid].EntryOff <= pcOff {
			lo = mid
		} else {
			hi = mid
		}
	}
	for lo+1 < len(ftab) && ftab[lo+1].EntryOff <= pcOff {
		lo++
	}
	if lo >= len(ftab)-1 {
		return len(ftab) - 2
	}
	return lo
}

// LookupFuncIndex mirrors runtime.findfunc's hot lookup: use the bucket and
// subbucket to jump near the target function, then linearly scan the remaining
// entries in that small range.
func LookupFuncIndex(ftab []FuncTabEntry, buckets []FindFuncBucket, pcOff uint32) int {
	if len(ftab) < 2 || len(buckets) == 0 {
		return -1
	}
	bucket := pcOff / FuncTabBucketSize
	if bucket >= uint32(len(buckets)) {
		return -1
	}
	subSize := FuncTabBucketSize / FindFuncSubbucket
	sub := (pcOff % FuncTabBucketSize) / subSize
	b := buckets[bucket]
	idx := int(b.Idx) + int(b.Subbuckets[sub])
	for idx+1 < len(ftab) && ftab[idx+1].EntryOff <= pcOff {
		idx++
	}
	if idx >= len(ftab)-1 {
		return len(ftab) - 2
	}
	return idx
}
