// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

//go:build go1.26

//llgo:skipall
package sync

import (
	"sync/atomic"
	"unsafe"
)

type hashFunc func(unsafe.Pointer, uintptr) uintptr
type equalFunc func(unsafe.Pointer, unsafe.Pointer) bool

const (
	nChildrenLog2 = 4
	nChildren     = 1 << nChildrenLog2
)

type node[K comparable, V any] struct {
	isEntry bool
}

type indirect[K comparable, V any] struct {
	node[K, V]
	dead     atomic.Bool
	mu       Mutex
	parent   *indirect[K, V]
	children [nChildren]atomic.Pointer[node[K, V]]
}

type entry[K comparable, V any] struct {
	node[K, V]
	overflow atomic.Pointer[entry[K, V]]
	key      K
	value    V
}

type hashTrieEntry[K comparable, V any] struct {
	key   K
	value V
}

// HashTrieMap is a compatibility implementation for llgo.
// It preserves the zero-value and concurrency semantics expected by sync.Map
// while using a mutex-protected key-to-index map for average O(1) lookups and
// a dense entry slice for efficient Range snapshots.
type HashTrieMap[K comparable, V any] struct {
	// Keep the leading field layout compatible with the upstream type so
	// imported stdlib code that still selects these fields by index can build.
	inited   atomic.Uint32
	initMu   Mutex
	root     atomic.Pointer[indirect[K, V]]
	keyHash  hashFunc
	valEqual equalFunc
	seed     uintptr

	mu      Mutex
	m       map[K]int
	entries []hashTrieEntry[K, V]
}

func (ht *HashTrieMap[K, V]) ensureMap() {
	if ht.m == nil {
		ht.m = make(map[K]int)
	}
}

func (ht *HashTrieMap[K, V]) Load(key K) (value V, ok bool) {
	ht.mu.Lock()
	defer ht.mu.Unlock()
	index, ok := ht.m[key]
	if ok {
		value = ht.entries[index].value
	}
	return
}

func (ht *HashTrieMap[K, V]) LoadOrStore(key K, value V) (result V, loaded bool) {
	ht.mu.Lock()
	defer ht.mu.Unlock()
	ht.ensureMap()
	if index, ok := ht.m[key]; ok {
		return ht.entries[index].value, true
	}
	ht.m[key] = len(ht.entries)
	ht.entries = append(ht.entries, hashTrieEntry[K, V]{key: key, value: value})
	return value, false
}

func (ht *HashTrieMap[K, V]) Store(key K, value V) {
	_, _ = ht.Swap(key, value)
}

func (ht *HashTrieMap[K, V]) Swap(key K, new V) (previous V, loaded bool) {
	ht.mu.Lock()
	defer ht.mu.Unlock()
	ht.ensureMap()
	if index, ok := ht.m[key]; ok {
		previous, loaded = ht.entries[index].value, true
		ht.entries[index].value = new
		return
	}
	ht.m[key] = len(ht.entries)
	ht.entries = append(ht.entries, hashTrieEntry[K, V]{key: key, value: new})
	return
}

func (ht *HashTrieMap[K, V]) CompareAndSwap(key K, old, new V) bool {
	ht.mu.Lock()
	defer ht.mu.Unlock()
	index, ok := ht.m[key]
	if !ok || !hashTrieValueEqual(ht.entries[index].value, old) {
		return false
	}
	ht.entries[index].value = new
	return true
}

func (ht *HashTrieMap[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	ht.mu.Lock()
	defer ht.mu.Unlock()
	index, loaded := ht.m[key]
	if loaded {
		value = ht.entries[index].value
		ht.deleteIndex(index)
	}
	return
}

func (ht *HashTrieMap[K, V]) Delete(key K) {
	_, _ = ht.LoadAndDelete(key)
}

func (ht *HashTrieMap[K, V]) CompareAndDelete(key K, old V) bool {
	ht.mu.Lock()
	defer ht.mu.Unlock()
	index, ok := ht.m[key]
	if !ok || !hashTrieValueEqual(ht.entries[index].value, old) {
		return false
	}
	ht.deleteIndex(index)
	return true
}

func (ht *HashTrieMap[K, V]) All() func(yield func(K, V) bool) {
	return func(yield func(K, V) bool) {
		entries := ht.snapshot()
		for _, entry := range entries {
			if !yield(entry.key, entry.value) {
				return
			}
		}
	}
}

func (ht *HashTrieMap[K, V]) Range(yield func(K, V) bool) {
	ht.All()(yield)
}

func (ht *HashTrieMap[K, V]) Clear() {
	ht.mu.Lock()
	ht.m = nil
	ht.entries = nil
	ht.mu.Unlock()
}

func (ht *HashTrieMap[K, V]) snapshot() []hashTrieEntry[K, V] {
	ht.mu.Lock()
	defer ht.mu.Unlock()
	if len(ht.entries) == 0 {
		return nil
	}
	entries := make([]hashTrieEntry[K, V], len(ht.entries))
	copy(entries, ht.entries)
	return entries
}

func (ht *HashTrieMap[K, V]) deleteIndex(index int) {
	last := len(ht.entries) - 1
	removed := ht.entries[index].key
	if index != last {
		moved := ht.entries[last]
		ht.entries[index] = moved
		// Non-reflexive keys cannot be looked up; assigning again adds an entry.
		if moved.key == moved.key {
			ht.m[moved.key] = index
		}
	}
	delete(ht.m, removed)
	var zero hashTrieEntry[K, V]
	ht.entries[last] = zero
	ht.entries = ht.entries[:last]
}

func hashTrieValueEqual[V any](a, b V) bool {
	// Match upstream sync.Map CompareAndSwap/CompareAndDelete behavior:
	// non-comparable values panic during equality comparison.
	return any(a) == any(b)
}
