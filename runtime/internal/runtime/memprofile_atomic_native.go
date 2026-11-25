//go:build llgo && !baremetal && !wasm

package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

//llgo:link memProfileCtlz64 llvm.ctlz
func memProfileCtlz64(x uint64, isZeroPoison bool) uint64 { return 0 }

const memProfileShards = 64

// Keep all size classes for one shard contiguous and padded to 256 bytes
// (two 128-byte cache lines) so hot small allocations (16 B..2048 B) share a
// single cache line per shard without false sharing across threads.
type memProfileShard struct {
	_       memProfileAlignment
	counts  [memProfileNumBuckets]uint64
	padding [5]uint64
}

var memProfileShardsData [memProfileShards]memProfileShard

func memProfileBucketIndex(size uintptr) int {
	if size <= 16 {
		return 0
	}
	if size >= memProfileMaxSizeClass {
		return memProfileNumBuckets - 1
	}
	return 64 - int(memProfileCtlz64(uint64(size-1)>>4, true))
}

// Hash a stack address with Fibonacci multiplication to spread both
// power-of-two (Linux) and guard-padded (macOS 0x84000) OS thread stacks
// across shards without invoking TLS accessors or initializing G.
func memProfileAddBucket(idx int) {
	var anchor byte
	id := uint64(uintptr(noescape(unsafe.Pointer(&anchor))))
	shard := (((id >> 14) ^ (id >> 21)) * 0x9e3779b97f4a7c15) >> 58
	atomic.AddRelaxed(&memProfileShardsData[shard].counts[idx], uint64(1))
}

func memProfileLoadBucket(idx int) uint64 {
	var total uint64
	for i := range memProfileShardsData {
		total += atomic.Load(&memProfileShardsData[i].counts[idx])
	}
	return total
}
