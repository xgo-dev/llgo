// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// llgo:skipall

package unique

import (
	"runtime"
	"sync"
	"unsafe"
	"weak"

	"github.com/goplus/llgo/runtime/abi"
)

var zero uintptr

// Handle is a globally unique identity for some value of type T.
//
// Two handles compare equal exactly if the two values used to create the handles
// would have also compared equal. The comparison of two handles is trivial and
// typically much more efficient than comparing the values used to create them.
type Handle[T comparable] struct {
	value *T
}

// Value returns a shallow copy of the T value that produced the Handle.
// Value is safe for concurrent use by multiple goroutines.
func (h Handle[T]) Value() T {
	return *h.value
}

// Make returns a globally unique handle for a value of type T. Handles
// are equal if and only if the values used to produce them are equal.
// Make is safe for concurrent use by multiple goroutines.
func Make[T comparable](value T) Handle[T] {
	typ := abi.TypeFor[T]()
	if typ.Size() == 0 {
		return Handle[T]{(*T)(unsafe.Pointer(&zero))}
	}
	ma, ok := loadUniqueMap(typ)
	if !ok {
		ensureSetupMake()
		ma = addUniqueMap[T](typ)
	}
	m := ma.(*uniqueMap[T])

	var (
		toInsert     *T
		toInsertWeak weak.Pointer[T]
	)
	newValue := func() (T, weak.Pointer[T]) {
		if toInsert == nil {
			toInsert = new(T)
			*toInsert = clone(value, &m.cloneSeq)
			toInsertWeak = weak.Make(toInsert)
		}
		return *toInsert, toInsertWeak
	}
	var ptr *T
	for {
		wp, ok := m.Load(value)
		if !ok {
			k, v := newValue()
			wp, _ = m.LoadOrStore(k, v)
		}
		ptr = wp.Value()
		if ptr != nil {
			break
		}
		m.CompareAndDelete(value, wp)
	}
	runtime.KeepAlive(toInsert)
	return Handle[T]{ptr}
}

var (
	uniqueMapsMu sync.Mutex
	uniqueMaps   map[*abi.Type]any

	// Lock ordering:
	// uniqueMapsMu is never held with cleanupMu during cleanup registration;
	// cleanup callbacks take cleanupMu before cleanupFuncsMu, then release
	// cleanupFuncsMu before running per-type map cleanup functions.
	cleanupMu      sync.Mutex
	cleanupFuncsMu sync.Mutex
	cleanupFuncs   []func()
	cleanupNotify  []func()
)

type uniqueMap[T comparable] struct {
	mu sync.Mutex
	m  map[T]weak.Pointer[T]
	cloneSeq
}

type uniqueEntry[T comparable] struct {
	key   T
	value weak.Pointer[T]
}

func addUniqueMap[T comparable](typ *abi.Type) *uniqueMap[T] {
	m := &uniqueMap[T]{
		m:        make(map[T]weak.Pointer[T]),
		cloneSeq: makeCloneSeq(typ),
	}
	a, loaded := loadOrStoreUniqueMap(typ, m)
	if !loaded {
		cleanupFuncsMu.Lock()
		cleanupFuncs = append(cleanupFuncs, func() {
			m.All()(func(key T, wp weak.Pointer[T]) bool {
				if wp.Value() == nil {
					m.CompareAndDelete(key, wp)
				}
				return true
			})
		})
		cleanupFuncsMu.Unlock()
	}
	return a.(*uniqueMap[T])
}

var (
	setupMakeMu sync.Mutex
	setupMade   bool
)

func ensureSetupMake() {
	setupMakeMu.Lock()
	if !setupMade {
		registerCleanup()
		setupMade = true
	}
	setupMakeMu.Unlock()
}

func loadUniqueMap(typ *abi.Type) (any, bool) {
	uniqueMapsMu.Lock()
	if uniqueMaps == nil {
		uniqueMapsMu.Unlock()
		return nil, false
	}
	v, ok := uniqueMaps[typ]
	uniqueMapsMu.Unlock()
	return v, ok
}

func loadOrStoreUniqueMap(typ *abi.Type, value any) (any, bool) {
	uniqueMapsMu.Lock()
	if uniqueMaps == nil {
		uniqueMaps = make(map[*abi.Type]any)
	}
	if existing, ok := uniqueMaps[typ]; ok {
		uniqueMapsMu.Unlock()
		return existing, true
	}
	uniqueMaps[typ] = value
	uniqueMapsMu.Unlock()
	return value, false
}

func (m *uniqueMap[T]) Load(key T) (weak.Pointer[T], bool) {
	m.mu.Lock()
	v, ok := m.m[key]
	m.mu.Unlock()
	if !ok {
		var zero weak.Pointer[T]
		return zero, false
	}
	return v, true
}

func (m *uniqueMap[T]) LoadOrStore(key T, value weak.Pointer[T]) (weak.Pointer[T], bool) {
	m.mu.Lock()
	if existing, ok := m.m[key]; ok {
		m.mu.Unlock()
		return existing, true
	}
	m.m[key] = value
	m.mu.Unlock()
	return value, false
}

func (m *uniqueMap[T]) CompareAndDelete(key T, old weak.Pointer[T]) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.m[key]
	if !ok || v != old {
		return false
	}
	delete(m.m, key)
	return true
}

func (m *uniqueMap[T]) All() func(func(T, weak.Pointer[T]) bool) {
	return func(yield func(T, weak.Pointer[T]) bool) {
		m.mu.Lock()
		items := make([]uniqueEntry[T], 0, len(m.m))
		for k, v := range m.m {
			items = append(items, uniqueEntry[T]{key: k, value: v})
		}
		m.mu.Unlock()
		for _, item := range items {
			if !yield(item.key, item.value) {
				return
			}
		}
	}
}

func registerCleanup() {
	runtime_registerUniqueMapCleanup(func() {
		cleanupMu.Lock()

		cleanupFuncsMu.Lock()
		cf := cleanupFuncs
		cleanupFuncsMu.Unlock()

		for _, f := range cf {
			f()
		}

		for _, f := range cleanupNotify {
			f()
		}
		cleanupNotify = nil

		cleanupMu.Unlock()
	})
}

//go:linkname runtime_registerUniqueMapCleanup
func runtime_registerUniqueMapCleanup(cleanup func())
