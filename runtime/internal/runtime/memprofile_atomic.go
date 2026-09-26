//go:build !baremetal

package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

type memProfileCounter = uint64
type memProfileBucketHead = unsafe.Pointer

func memProfileAddObject(p *memProfileCounter) {
	atomic.Add(p, memProfileCounter(1))
}

func memProfileAddN(p *memProfileCounter, n uint64) {
	atomic.Add(p, memProfileCounter(n))
}

func memProfileLoadObjects(p *memProfileCounter) memProfileCounter {
	return atomic.Load(p)
}

func memProfileLoadBucket(p *memProfileBucketHead) *memStackBucket {
	return (*memStackBucket)(atomic.Load(p))
}

func memProfileStoreBucket(p *memProfileBucketHead, b *memStackBucket) {
	atomic.Store(p, unsafe.Pointer(b))
}
