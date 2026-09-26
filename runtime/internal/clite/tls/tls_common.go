//go:build llgo && !baremetal

/*
 * Copyright (c) 2025 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package tls provides generic storage backed by the host thread-local
// storage API. Native GC builds use scanned, uncollectable BDWGC slots;
// linear-memory wasm GC builds use explicit TinyGo GC roots. Both keep Go
// pointers in host TLS visible until the thread-local destructor runs.
// Builds without GC use calloc/free. Baremetal builds and builds without llgo
// use no-op handles.
//
// Basic usage:
//
//	h := tls.Alloc[int](nil)
//	h.Set(42)
//	val := h.Get() // returns 42
//
// With destructor:
//
//	h := tls.Alloc[*Resource](func(r **Resource) {
//	    if r != nil && *r != nil {
//	        (*r).Close()
//	    }
//	})
//
// Build tags:
//   - llgo && !baremetal && !wasm && !nogc: GC-managed slots via BDWGC
//   - llgo && !baremetal && wasm && llgo.wasm.gc.linear && !nogc: Explicit GC roots
//   - llgo && !baremetal && (nogc || (wasm && !llgo.wasm.gc.linear)): calloc/free
//   - !llgo || baremetal: No-op handles
package tls

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/thread"
)

type Handle[T any] struct {
	key        thread.Key
	destructor func(*T)
}

// Alloc creates a handle backed by the host thread-local storage API.
func Alloc[T any](destructor func(*T)) Handle[T] {
	var key thread.Key
	if ret := key.Create(thread.KeyDestructor(slotDestructor[T])); ret != 0 {
		c.Fprintf(c.Stderr, c.Str("tls: thread-local key creation failed (error=%d)\n"), ret)
		panic("tls: failed to create thread local storage key")
	}
	return Handle[T]{key: key, destructor: destructor}
}

// Get returns the value stored in the current thread's slot.
func (h Handle[T]) Get() T {
	if ptr := h.key.Get(); ptr != nil {
		return (*slot[T])(ptr).value
	}
	var zero T
	return zero
}

// Set stores v in the current thread's slot, creating it if necessary.
func (h Handle[T]) Set(v T) {
	s := h.ensureSlot()
	s.value = v
}

// Clear zeroes the current thread's slot value without freeing the slot.
func (h Handle[T]) Clear() {
	if ptr := h.key.Get(); ptr != nil {
		s := (*slot[T])(ptr)
		var zero T
		s.value = zero
	}
}

func (h Handle[T]) ensureSlot() *slot[T] {
	if ptr := h.key.Get(); ptr != nil {
		return (*slot[T])(ptr)
	}
	size := unsafe.Sizeof(slot[T]{})
	mem := allocSlot(size)
	if mem == nil {
		panic("tls: failed to allocate thread slot")
	}
	s := (*slot[T])(mem)
	s.destructor = h.destructor
	// Collector allocation can run finalizers that re-enter this handle.
	if existing := h.key.Get(); existing != nil {
		freeSlot(mem)
		return (*slot[T])(existing)
	}
	if ret := h.key.Set(mem); ret != 0 {
		freeSlot(mem)
		c.Fprintf(c.Stderr, c.Str("tls: thread-local value installation failed (error=%d)\n"), ret)
		panic("tls: failed to set thread local storage value")
	}
	return s
}

func slotDestructor[T any](ptr c.Pointer) {
	s := (*slot[T])(ptr)
	if s == nil {
		return
	}
	if s.destructor != nil {
		s.destructor(&s.value)
	}
	var zero T
	s.value = zero
	s.destructor = nil
	freeSlot(ptr)
}
