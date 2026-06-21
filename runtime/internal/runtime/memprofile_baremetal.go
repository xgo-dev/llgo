//go:build baremetal

package runtime

type memProfileCounter = uintptr

func memProfileAddObject(p *memProfileCounter) {
	*p = *p + 1
}

func memProfileLoadObjects(p *memProfileCounter) memProfileCounter {
	return *p
}
