//go:build llgo && !baremetal && !wasm

package runtime

import "github.com/xgo-dev/llgo/runtime/internal/sync/atomic"

const memProfileShards = 32

type memProfileCounterShard struct {
	_       memProfileAlignment
	value   uint64
	padding [15]uint64
}

type memProfileCounter struct {
	shards [memProfileShards]memProfileCounterShard
}

// Hash the physical-thread slot without initializing or dereferencing a G.
// Bootstrap/foreign contexts and hash collisions retain exact atomic counting.
func memProfileAddObject(p *memProfileCounter) {
	id := uint64(currentG)
	shard := (id>>4 ^ id>>11 ^ id>>19 ^ id>>27) & (memProfileShards - 1)
	atomic.Add(&p.shards[shard].value, uint64(1))
}

func memProfileLoadObjects(p *memProfileCounter) uint64 {
	var total uint64
	for i := range p.shards {
		total += atomic.Load(&p.shards[i].value)
	}
	return total
}
