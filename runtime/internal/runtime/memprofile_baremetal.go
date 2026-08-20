//go:build baremetal

package runtime

import "unsafe"

type memProfileCounter = uintptr
type memProfileBucketHead = unsafe.Pointer

// Bare-metal runtimes have a single execution context and must not introduce
// native TLS relocations.
var memProfileState memProfileThreadState

func memProfileAddN(p *memProfileCounter, n uint64) {
	*p += memProfileCounter(n)
}

func memProfileAddObject(p *memProfileCounter) {
	*p = *p + 1
}

func memProfileLoadObjects(p *memProfileCounter) memProfileCounter {
	return *p
}

func memProfileLoadBucket(p *memProfileBucketHead) *memStackBucket {
	return (*memStackBucket)(*p)
}

func memProfileStoreBucket(p *memProfileBucketHead, b *memStackBucket) {
	*p = unsafe.Pointer(b)
}
