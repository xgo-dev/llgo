//go:build !nogc && !baremetal

/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/bdwgc"
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

// AllocU allocates uninitialized memory and returns a non-nil pointer or panics.
// A zero-byte request still allocates at least one byte.
func AllocU(size uintptr) unsafe.Pointer {
	n := size
	if n == 0 {
		n = 1
	}
	ret := bdwgc.Malloc(n)
	if ret == nil {
		panic("out of memory")
	}
	recordMemProfileAlloc(size)
	return ret
}

// AllocZ allocates zero-initialized memory.
func AllocZ(size uintptr) unsafe.Pointer {
	ret := AllocU(size)
	c.Memset(ret, 0, size)
	return ret
}

func AllocRoot(size uintptr) unsafe.Pointer {
	if size == 0 {
		size = 1
	}
	ret := bdwgc.MallocUncollectable(size)
	if ret == nil {
		panic("out of memory")
	}
	return ret
}

func FreeRoot(ptr unsafe.Pointer) {
	bdwgc.Free(ptr)
}

type entry struct {
	fn    func()         // cleanup func
	prev  unsafe.Pointer // prev cleanup func ptr
	slot  *cleanupSlot
	id    uint64 // non-zero for a Cleanup handle
	state int32
}

const (
	cleanupActive int32 = iota
	cleanupStopped
	cleanupRunning
	cleanupDone
)

type cleanupSlot struct {
	entry      unsafe.Pointer
	nextFree   unsafe.Pointer
	index      uint32
	generation uint32
}

// cleanupSlots keeps callback entries reachable without storing an object
// pointer in runtime.Cleanup. Slots are reused only after BDWGC invokes the
// finalizer, and the generation in each id makes stale Cleanup values harmless.
var cleanupSlots struct {
	once psync.Once
	mu   psync.Mutex
	all  []*cleanupSlot
	free unsafe.Pointer
}

func initCleanupSlots() {
	cleanupSlots.mu.Init(nil)
}

// freeCleanupSlot is called from BDWGC finalizers. It must not allocate or
// acquire a lock: BDWGC may invoke another finalizer while Go code is in a
// collector allocation or map operation.
func freeCleanupSlot(e *entry) {
	slot := e.slot
	if _, ok := atomic.CompareAndExchange(&slot.entry, unsafe.Pointer(e), nil); !ok {
		return
	}
	for {
		head := atomic.Load(&cleanupSlots.free)
		atomic.Store(&slot.nextFree, head)
		if _, ok := atomic.CompareAndExchange(&cleanupSlots.free, head, unsafe.Pointer(slot)); ok {
			return
		}
	}
}

// popCleanupSlot runs with cleanupSlots.mu held. Finalizers publish freed slots
// concurrently, so the free-list head still requires atomic operations.
func popCleanupSlot() *cleanupSlot {
	for {
		head := atomic.Load(&cleanupSlots.free)
		if head == nil {
			return nil
		}
		slot := (*cleanupSlot)(head)
		next := atomic.Load(&slot.nextFree)
		if _, ok := atomic.CompareAndExchange(&cleanupSlots.free, head, next); ok {
			atomic.Store(&slot.nextFree, nil)
			return slot
		}
	}
}

func newCancelableCleanup(cleanup func()) *entry {
	cleanupSlots.once.Do(initCleanupSlots)
	cleanupSlots.mu.Lock()
	slot := popCleanupSlot()
	if slot == nil {
		if uint64(len(cleanupSlots.all)) >= uint64(^uint32(0)) {
			cleanupSlots.mu.Unlock()
			panic("runtime: too many pending cleanups")
		}
		slot = &cleanupSlot{index: uint32(len(cleanupSlots.all))}
		cleanupSlots.all = append(cleanupSlots.all, slot)
	}
	slot.generation++
	if slot.generation == 0 {
		slot.generation++
	}
	id := uint64(slot.generation)<<32 | uint64(slot.index+1)
	e := &entry{fn: cleanup, slot: slot, id: id}
	atomic.Store(&slot.entry, unsafe.Pointer(e))
	cleanupSlots.mu.Unlock()
	return e
}

func finalizer(ptr unsafe.Pointer, cb unsafe.Pointer) {
	e := (*entry)(cb)
	if ptr := atomic.Load(&e.prev); ptr != nil {
		(*(*func())(ptr))()
	}
	if e.id == 0 {
		if atomic.Load(&e.state) != cleanupStopped {
			e.fn()
		}
		return
	}
	_, run := atomic.CompareAndExchange(&e.state, cleanupActive, cleanupRunning)
	if run {
		e.fn()
	}
	atomic.Store(&e.state, cleanupDone)
	freeCleanupSlot(e)
}

func registerCleanupPtr(ptr unsafe.Pointer, e *entry) {
	var oldFn bdwgc.FinalizerFunc
	var oldCb unsafe.Pointer
	bdwgc.RegisterFinalizer(ptr, finalizer, unsafe.Pointer(e), &oldFn, &oldCb)
	if oldCb != nil {
		n := uintptr(ptr) ^ 0xffff // hides the pointer from escape analysis
		fn := func() {
			oldFn((unsafe.Pointer)(n^0xffff), oldCb)
		}
		atomic.Store(&e.prev, unsafe.Pointer(&fn))
	}
}

// AddCleanupPtr attaches a cleanup function to ptr. Some time after ptr is no longer
// reachable, the runtime will call cleanup().
func AddCleanupPtr(ptr unsafe.Pointer, cleanup func()) (cancel func()) {
	e := &entry{fn: cleanup}
	registerCleanupPtr(ptr, e)
	return func() {
		atomic.Store(&e.state, cleanupStopped)
	}
}

// AddCancelableCleanupPtr registers a cleanup and returns a stable, pointer-free
// identifier suitable for runtime.Cleanup's Go-compatible representation.
func AddCancelableCleanupPtr(ptr unsafe.Pointer, cleanup func()) uint64 {
	e := newCancelableCleanup(cleanup)
	registerCleanupPtr(ptr, e)
	return e.id
}

// StopCleanupPtr cancels a pending cleanup. If its finalizer has already
// claimed the entry, Stop has no effect, matching runtime.Cleanup.Stop.
func StopCleanupPtr(id uint64) {
	if id == 0 {
		return
	}
	cleanupSlots.once.Do(initCleanupSlots)
	index := uint64(uint32(id) - 1)
	cleanupSlots.mu.Lock()
	if index < uint64(len(cleanupSlots.all)) {
		slot := cleanupSlots.all[index]
		if e := (*entry)(atomic.Load(&slot.entry)); e != nil && e.id == id {
			atomic.CompareAndExchange(&e.state, cleanupActive, cleanupStopped)
		}
	}
	cleanupSlots.mu.Unlock()
}
