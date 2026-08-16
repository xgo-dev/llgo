// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sync

import (
	"sync/atomic"
	"unsafe"
)

// A Pool is a set of temporary objects that may be individually saved and
// retrieved.
//
// Any item stored in the Pool may be removed automatically at any time without
// notification. If the Pool holds the only reference when this happens, the
// item might be deallocated.
//
// A Pool is safe for use by multiple goroutines simultaneously.
//
// Pool's purpose is to cache allocated but unused items for later reuse,
// relieving pressure on the garbage collector. That is, it makes it easy to
// build efficient, thread-safe free lists. However, it is not suitable for all
// free lists.
//
// An appropriate use of a Pool is to manage a group of temporary items
// silently shared among and potentially reused by concurrent independent
// clients of a package. Pool provides a way to amortize allocation overhead
// across many clients.
//
// An example of good use of a Pool is in the fmt package, which maintains a
// dynamically-sized store of temporary output buffers. The store scales under
// load (when many goroutines are actively printing) and shrinks when
// quiescent.
//
// On the other hand, a free list maintained as part of a short-lived object is
// not a suitable use for a Pool, since the overhead does not amortize well in
// that scenario. It is more efficient to have such objects implement their own
// free list.
//
// A Pool must not be copied after first use.
//
// In the terminology of [the Go memory model], a call to Put(x) “synchronizes before”
// a call to [Pool.Get] returning that same value x.
// Similarly, a call to New returning x “synchronizes before”
// a call to Get returning that same value x.
//
// [the Go memory model]: https://go.dev/ref/mem
type Pool struct {
	noCopy noCopy

	// Keep the standard-library field sequence: the overlay is compiled
	// against the exported sync.Pool type data even though LLGo's 1:1 backend
	// uses one dynamically allocated TLS local rather than a [P] array.
	local     unsafe.Pointer // local fixed-size per-P pool, actual type is [P]poolLocal
	localSize uintptr        // size of the local array

	victim     unsafe.Pointer // local from previous cycle
	victimSize uintptr        // size of victims array

	// New optionally specifies a function to generate
	// a value when Get would otherwise return nil.
	// It may not be changed concurrently with calls to Get.
	New  func() any
	once Once
}

// Local per-P Pool appendix.
type poolLocalInternal struct {
	private any       // Can be used only by the respective P.
	shared  poolChain // Local P can pushHead/popHead; any P can popTail.
}

type poolLocal struct {
	poolLocalInternal

	// Prevents false sharing on widespread platforms with
	// 128 mod (cache line size) = 0 .
	pad [128 - unsafe.Sizeof(poolLocalInternal{})%128]byte
}

// Put adds x to the pool.
func (p *Pool) Put(x any) {
	if x == nil {
		return
	}
	l, _ := p.pin()
	if l.private == nil {
		l.private = x
	} else {
		l.shared.pushHead(x)
	}
}

// Get selects an arbitrary item from the [Pool], removes it from the
// Pool, and returns it to the caller.
// Get may choose to ignore the pool and treat it as empty.
// Callers should not assume any relation between values passed to [Pool.Put] and
// the values returned by Get.
//
// If Get would otherwise return nil and p.New is non-nil, Get returns
// the result of calling p.New.
func (p *Pool) Get() any {
	l, _ := p.pin()
	x := l.private
	l.private = nil
	if x == nil {
		x, _ = l.shared.popHead()
		if x == nil {
			x = p.getSlow(0)
		}
	}
	if x == nil && p.New != nil {
		x = p.New()
	}
	return x
}

// pin returns the Pool cache for the current execution resource. LLGo's
// current 1:1 P/M/thread binding lets the dynamic pthread TLS slot model the
// per-P lifetime; the returned P id is therefore always zero.
func (p *Pool) pin() (*poolLocal, int) {
	// Check whether p is nil to get a panic.
	// Otherwise the nil dereference happens while the m is pinned,
	// causing a fatal error rather than a panic.
	if p == nil {
		panic("nil Pool")
	}

	if ptr := atomic.LoadPointer(&p.local); ptr != nil {
		l := (*poolLocal)(runtime_poolLocalGet(ptr))
		if l == nil {
			l = &poolLocal{}
			runtime_poolLocalSet(ptr, unsafe.Pointer(l))
		}
		return l, 0
	}

	return p.pinSlow()
}

func (p *Pool) pinSlow() (*poolLocal, int) {
	p.once.Do(func() {
		atomic.StorePointer(&p.local, runtime_poolLocalAlloc(&p.victim))
	})
	l := &poolLocal{}
	runtime_poolLocalSet(p.local, unsafe.Pointer(l))
	return l, 0
}

func (p *Pool) getSlow(pid int) any {
	if ptr := atomic.LoadPointer(&p.victim); ptr != nil {
		l := (*poolLocal)(ptr)
		if x, _ := l.shared.popTail(); x != nil {
			return x
		}
		atomic.StorePointer(&p.victim, nil)
	}
	return nil
}

// The standard cleanup operates on a per-P array stored in Pool.local. LLGo
// stores a dynamic TLS handle there instead and releases its locals at thread
// exit, so the runtime GC hook must not reinterpret that pointer.
func poolCleanup() {}

// Implemented in runtime. Keeping the dynamic TLS implementation behind these
// hooks lets the patched standard package retain its original dependency graph.
func runtime_poolLocalAlloc(victim *unsafe.Pointer) unsafe.Pointer
func runtime_poolLocalGet(handle unsafe.Pointer) unsafe.Pointer
func runtime_poolLocalSet(handle, local unsafe.Pointer)
