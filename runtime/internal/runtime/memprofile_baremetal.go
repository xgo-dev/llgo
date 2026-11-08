//go:build baremetal

package runtime

// Baremetal accounting uses a non-atomic native-word counter.
type memProfileCounter = uintptr

func memProfileAddObject(p *memProfileCounter) {
	*p = *p + 1
}

func memProfileLoadObjects(p *memProfileCounter) uint64 {
	return uint64(*p)
}
