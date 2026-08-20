//go:build !llgo || baremetal || wasm

package runtime

func MemProfilePause() uintptr {
	return 0
}

func MemProfileResume(uintptr) {
}

// Legacy targets do not yet expose the physical native stack walk used by the
// sampled profiler. Keep their existing size-class accounting separate so the
// native allocator hot path has no mode dispatch.
func recordMemProfileAlloc(size uintptr) {
	if size == 0 {
		return
	}
	sizeClass := memProfileSizeClass(size)
	for i := range memProfileBuckets {
		b := &memProfileBuckets[i]
		if b.size == sizeClass {
			memProfileAddObject(&b.objects)
			return
		}
	}
}
