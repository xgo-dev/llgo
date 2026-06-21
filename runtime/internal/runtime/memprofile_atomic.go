//go:build !baremetal

package runtime

import "github.com/goplus/llgo/runtime/internal/clite/sync/atomic"

type memProfileCounter = uint64

func memProfileAddObject(p *memProfileCounter) {
	atomic.Add(p, memProfileCounter(1))
}

func memProfileLoadObjects(p *memProfileCounter) memProfileCounter {
	return atomic.Load(p)
}
