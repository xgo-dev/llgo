//go:build llgo && !baremetal && !wasm

package runtime

import "github.com/xgo-dev/llgo/runtime/internal/sync/atomic"

const memProfileShards = 32

type memProfileCounterShard struct {
	_     memProfileAlignment
	value uint64
	// Separate counters on hosts with 128-byte cache lines.
	padding [15]uint64
}

type memProfileCounter struct {
	shards [memProfileShards]memProfileCounterShard
}

// Hash the numeric current-G TLS value without initializing or dereferencing G.
// Goroutine switches may select different shards.
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
