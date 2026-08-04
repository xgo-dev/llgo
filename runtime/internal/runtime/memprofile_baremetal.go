//go:build baremetal

package runtime

type memProfileCounter = uintptr

func memProfileAddN(p *memProfileCounter, n uint64) {
	*p += memProfileCounter(n)
}

func memProfileAddObject(p *memProfileCounter) {
	*p = *p + 1
}

func memProfileLoadObjects(p *memProfileCounter) memProfileCounter {
	return *p
}
