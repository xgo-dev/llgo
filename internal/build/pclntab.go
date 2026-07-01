//go:build !llgo
// +build !llgo

package build

import (
	"fmt"

	llvm "github.com/xgo-dev/llvm"
)

const (
	pclnMinFuncSize       = uint32(16)
	pclnFuncTabBucketSize = uint32(256) * pclnMinFuncSize
	pclnFindFuncSubbucket = 16
)

type pclnFuncTabEntry struct {
	entryOff uint32
	funcOff  uint32
}

type pclnFindFuncBucket struct {
	idx        uint32
	subbuckets [pclnFindFuncSubbucket]uint8
}

func buildPCLnFindFuncBuckets(ftab []pclnFuncTabEntry, textSize uint32) ([]pclnFindFuncBucket, error) {
	if textSize == 0 {
		return nil, nil
	}
	if len(ftab) < 2 {
		return nil, fmt.Errorf("pclntab ftab needs at least one function and one sentinel")
	}
	for i := 1; i < len(ftab); i++ {
		if ftab[i].entryOff <= ftab[i-1].entryOff {
			return nil, fmt.Errorf("pclntab ftab entries must be strictly increasing")
		}
	}
	if ftab[0].entryOff != 0 {
		return nil, fmt.Errorf("pclntab first entry offset must be zero")
	}
	if ftab[len(ftab)-1].entryOff < textSize {
		return nil, fmt.Errorf("pclntab sentinel offset %d below text size %d", ftab[len(ftab)-1].entryOff, textSize)
	}

	nbuckets := int((textSize + pclnFuncTabBucketSize - 1) / pclnFuncTabBucketSize)
	buckets := make([]pclnFindFuncBucket, nbuckets)
	subSize := pclnFuncTabBucketSize / pclnFindFuncSubbucket
	for b := range buckets {
		bucketStart := uint32(b) * pclnFuncTabBucketSize
		baseIdx := pclnFuncIndexForPC(ftab, bucketStart)
		buckets[b].idx = uint32(baseIdx)
		for s := 0; s < pclnFindFuncSubbucket; s++ {
			pc := bucketStart + uint32(s)*subSize
			if pc >= textSize {
				pc = textSize - 1
			}
			subIdx := pclnFuncIndexForPC(ftab, pc)
			delta := subIdx - baseIdx
			if delta < 0 || delta > 255 {
				return nil, fmt.Errorf("pclntab subbucket delta overflow: bucket=%d subbucket=%d delta=%d", b, s, delta)
			}
			buckets[b].subbuckets[s] = uint8(delta)
		}
	}
	return buckets, nil
}

func pclnFuncIndexForPC(ftab []pclnFuncTabEntry, pcOff uint32) int {
	lo, hi := 0, len(ftab)-1 // last entry is the sentinel.
	for lo+1 < hi {
		mid := int(uint(lo+hi) >> 1)
		if ftab[mid].entryOff <= pcOff {
			lo = mid
		} else {
			hi = mid
		}
	}
	for lo+1 < len(ftab) && ftab[lo+1].entryOff <= pcOff {
		lo++
	}
	if lo >= len(ftab)-1 {
		return len(ftab) - 2
	}
	return lo
}

func pclnLookupFuncIndex(ftab []pclnFuncTabEntry, buckets []pclnFindFuncBucket, pcOff uint32) int {
	if len(ftab) < 2 || len(buckets) == 0 {
		return -1
	}
	bucket := pcOff / pclnFuncTabBucketSize
	if bucket >= uint32(len(buckets)) {
		return -1
	}
	subSize := pclnFuncTabBucketSize / pclnFindFuncSubbucket
	sub := (pcOff % pclnFuncTabBucketSize) / subSize
	b := buckets[bucket]
	idx := int(b.idx) + int(b.subbuckets[sub])
	for idx+1 < len(ftab) && ftab[idx+1].entryOff <= pcOff {
		idx++
	}
	if idx >= len(ftab)-1 {
		return len(ftab) - 2
	}
	return idx
}

func emitPCLnFindFuncBuckets(mod llvm.Module, symbol string, buckets []pclnFindFuncBucket) llvm.Value {
	ctx := mod.Context()
	i8Type := ctx.Int8Type()
	i32Type := ctx.Int32Type()
	subType := llvm.ArrayType(i8Type, pclnFindFuncSubbucket)
	bucketType := ctx.StructType([]llvm.Type{i32Type, subType}, false)
	arrayType := llvm.ArrayType(bucketType, len(buckets))
	values := make([]llvm.Value, 0, len(buckets))
	for _, bucket := range buckets {
		subs := make([]llvm.Value, 0, len(bucket.subbuckets))
		for _, sub := range bucket.subbuckets {
			subs = append(subs, llvm.ConstInt(i8Type, uint64(sub), false))
		}
		values = append(values, llvm.ConstNamedStruct(bucketType, []llvm.Value{
			llvm.ConstInt(i32Type, uint64(bucket.idx), false),
			llvm.ConstArray(i8Type, subs),
		}))
	}
	global := llvm.AddGlobal(mod, arrayType, symbol)
	global.SetInitializer(llvm.ConstArray(bucketType, values))
	global.SetGlobalConstant(true)
	global.SetUnnamedAddr(true)
	global.SetAlignment(4)
	return global
}
