//go:build !baremetal

package runtime

import "sync/atomic"

// Uint64 carries the alignment required for atomic access on 32-bit targets,
// including when the counter follows a uintptr in a bucket or array element.
type memProfileCounter = atomic.Uint64

func memProfileAddObject(p *memProfileCounter) {
	p.Add(1)
}

func memProfileLoadObjects(p *memProfileCounter) uint64 {
	return p.Load()
}
