// Package wordmap provides compact pointer and encoded-address registries.
package wordmap

import "unsafe"

type key interface {
	~uintptr | unsafe.Pointer
}

type entry[K key, V any] struct {
	key   K
	value V
}

// noCopy lets go vet diagnose copies of an initialized Map.
type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

// Map is an unsynchronized table of nonzero machine-word keys. Its zero value
// is ready to use. Pointer keys remain pointers so the collector retains them.
// Encoded addresses can use uintptr keys without introducing pointer slots.
// A Map must not be copied after first use.
type Map[K key, V any] struct {
	_       noCopy
	entries []entry[K, V]
	count   int
}

func hash[K key](k K) uintptr {
	h := uintptr(k)
	h ^= h >> 32
	h ^= h >> 16
	h *= 0x45d9f3b
	return h ^ (h >> 16)
}

func (m *Map[K, V]) Get(k K) V {
	var zeroKey K
	if k != zeroKey && len(m.entries) != 0 {
		mask := len(m.entries) - 1
		for i := int(hash(k) & uintptr(mask)); ; i = (i + 1) & mask {
			e := &m.entries[i]
			if e.key == k {
				return e.value
			}
			if e.key == zeroKey {
				break
			}
		}
	}
	var zero V
	return zero
}

// Set stores v for k. It panics if k is the zero key.
func (m *Map[K, V]) Set(k K, v V) {
	var zeroKey K
	if k == zeroKey {
		panic("wordmap: zero key")
	}
	if !m.insert(k, v) {
		n := len(m.entries) * 2
		if n == 0 {
			n = 8
		}
		old := m.entries
		m.entries = make([]entry[K, V], n)
		m.count = 0
		for _, e := range old {
			if e.key != zeroKey {
				m.insert(e.key, e.value)
			}
		}
		m.insert(k, v)
	}
}

// insert returns false when a new key requires growth.
func (m *Map[K, V]) insert(k K, v V) bool {
	if len(m.entries) == 0 {
		return false
	}
	var zeroKey K
	mask := len(m.entries) - 1
	for i := int(hash(k) & uintptr(mask)); ; i = (i + 1) & mask {
		e := &m.entries[i]
		if e.key == zeroKey {
			if (m.count+1)*4 > len(m.entries)*3 {
				return false
			}
			m.count++
			e.key = k
		}
		if e.key == k {
			e.value = v
			return true
		}
	}
}

func (m *Map[K, V]) Delete(k K) {
	var zeroKey K
	if k == zeroKey || len(m.entries) == 0 {
		return
	}
	mask := len(m.entries) - 1
	i := int(hash(k) & uintptr(mask))
	for m.entries[i].key != k {
		if m.entries[i].key == zeroKey {
			return
		}
		i = (i + 1) & mask
	}
	m.count--
	// Move later entries across the hole only when their probe crossed it.
	// This also handles clusters wrapping around the end of the table.
	hole := i
	for i = (i + 1) & mask; m.entries[i].key != zeroKey; i = (i + 1) & mask {
		home := int(hash(m.entries[i].key) & uintptr(mask))
		if (i-home)&mask >= (i-hole)&mask {
			m.entries[hole] = m.entries[i]
			hole = i
		}
	}
	var zero entry[K, V]
	m.entries[hole] = zero
}

// Range visits each entry. The callback must not modify the table.
func (m *Map[K, V]) Range(f func(K, V)) {
	var zeroKey K
	for _, e := range m.entries {
		if e.key != zeroKey {
			f(e.key, e.value)
		}
	}
}
