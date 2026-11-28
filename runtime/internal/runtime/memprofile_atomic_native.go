//go:build llgo && !baremetal && !wasm

package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

//llgo:link memProfileCtlz64 llvm.ctlz
func memProfileCtlz64(x uint64, isZeroPoison bool) uint64 { return 0 }

const memProfileShards = 64

// Keep all size classes for one shard contiguous and padded to 256 bytes to
// avoid false sharing between shards while keeping hot small-allocation
// buckets (16 B..2048 B) close together. memProfileAlignment guarantees 8-byte
// alignment for 64-bit atomics on 32-bit targets.
type memProfileShard struct {
	_       memProfileAlignment
	counts  [memProfileNumBuckets]uint64
	padding [5]uint64
}

// memProfileShardsData is emitted with 256-byte base alignment matching
// memProfileShard's 256-byte stride (memProfileNumBuckets*8 + padding*8),
// ensuring adjacent shards never share a 64-byte or 128-byte cache line.
// The compiler overrides this global's alignment via
// runtimeMemProfileShardsSym and runtimeMemProfileShardsAlign in ssa/decl.go;
// update both when renaming memProfileShardsData or changing memProfileShard's
// stride.
var memProfileShardsData [memProfileShards]memProfileShard

// memProfileShardsBaseAddr exposes the linked base address for native layout tests.
func memProfileShardsBaseAddr() uintptr {
	return uintptr(unsafe.Pointer(&memProfileShardsData))
}

func memProfileBucketIndex(size uintptr) int {
	if size <= 16 {
		return 0
	}
	if size >= memProfileMaxSizeClass {
		return memProfileNumBuckets - 1
	}
	return 64 - int(memProfileCtlz64(uint64(size-1)>>4, true))
}

// memProfileShardIndex hashes a stack address with Fibonacci multiplication to
// spread both power-of-two (Linux) and guard-padded (macOS 0x84000) OS thread
// stacks across shards without invoking TLS accessors or initializing G.
func memProfileShardIndex(id uint64) uint64 {
	return (((id >> 14) ^ (id >> 21)) * 0x9e3779b97f4a7c15) >> 58
}

func memProfileAddBucket(idx int) {
	var anchor byte
	id := uint64(uintptr(noescape(unsafe.Pointer(&anchor))))
	shard := memProfileShardIndex(id)
	atomic.AddRelaxed(&memProfileShardsData[shard].counts[idx], uint64(1))
}

func memProfileLoadBucket(idx int) uint64 {
	var total uint64
	for i := range memProfileShardsData {
		total += atomic.Load(&memProfileShardsData[i].counts[idx])
	}
	return total
}
