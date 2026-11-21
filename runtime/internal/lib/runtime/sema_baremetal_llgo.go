//go:build llgo && baremetal && tinygo.riscv32 && !nogc

/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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

// Fiber-backed semaphore and notify list for the baremetal RV32 fiber
// scheduler (build tag: llgo && baremetal && tinygo.riscv32 && !nogc — the
// esp32c3/esp32s31 fiber-scheduler targets), mirroring sema_wasm_single_llgo.go.
// One hart runs every G cooperatively, so contention never needs memory
// synchronization: acquire parks the current G through a SchedulerWaiter and
// release makes a queued waiter runnable, the same primitives the channel
// implementation uses. stdlib sync.Mutex, RWMutex and WaitGroup, stdlib
// sync.Once (Mutex-based on Go 1.24+), and sync.Cond through the notify list
// all resolve here.
//
// Every other baremetal combination — nogc, non-llgo RV32, xtensa,
// cortex-m — keeps the pthread-backed fallback in sema_cond_llgo.go and
// sema_llgo.go, whose pthread_once is a libnosys stub that never runs its
// callback on baremetal: a known, accepted limitation, unchanged by this
// file.
//
// Synctest bubble boundary on this backend: Cond waits (notifyList) record a
// SynctestWaiter here and participate in bubbles, and durable WaitGroup waits
// keep using synctest_sema_llgo.go, whose durable path still initializes its
// state through psync.Once — pthread_once is a stub on baremetal, so the
// bubble + durable WaitGroup combination is unusable here (outside a bubble
// its fast path never touches it). The only user-reachable bubble entry is
// testing/synctest.Test, whose testing→flag→fmt import chain exceeds the
// C3-basic IRAM budget, so no emu fixture covers the bubble paths on this
// backend; the wiring matches sema_wasm_single_llgo.go, which carries the
// regression coverage.
//
// Waiters queue strictly FIFO: the cooperative scheduler parks waiters in
// call order and there is no fast-path spinning to favor, so the lifo
// handoff distinction of the native semaphore is immaterial on this backend.
// Parked stacks stay reachable the same way channel parking keeps them: the
// collector's StackRootsHook scans every parked fiber from its saved frame.

import (
	"sync/atomic"
	"unsafe"

	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
	"github.com/xgo-dev/llgo/runtime/internal/wordmap"
)

type fiberWaiter struct {
	synctest *llruntime.SynctestWaiter
	next     *fiberWaiter
	waiter   llruntime.SchedulerWaiter
	ticket   uint32
}

type fiberWaitQueue struct {
	head *fiberWaiter
	tail *fiberWaiter
}

// Reuse waiter records after their G resumes, matching the sudog cache role.
// Single-hart cooperative scheduling needs no lock around the free list.
var fiberWaiterFree *fiberWaiter

func acquireFiberWaiter(ticket uint32) *fiberWaiter {
	w := fiberWaiterFree
	if w == nil {
		w = new(fiberWaiter)
	} else {
		fiberWaiterFree = w.next
	}
	w.next = nil
	w.waiter = llruntime.CurrentSchedulerWaiter()
	w.ticket = ticket
	return w
}

func releaseFiberWaiter(w *fiberWaiter) {
	w.synctest = nil
	w.waiter = llruntime.SchedulerWaiter{}
	w.ticket = 0
	w.next = fiberWaiterFree
	fiberWaiterFree = w
}

func (q *fiberWaitQueue) push(w *fiberWaiter) {
	if q.tail == nil {
		q.head = w
	} else {
		q.tail.next = w
	}
	q.tail = w
}

func (q *fiberWaitQueue) pop() *fiberWaiter {
	w := q.head
	if w == nil {
		return nil
	}
	w.synctest.CheckWake()
	q.head = w.next
	if q.head == nil {
		q.tail = nil
	}
	w.next = nil
	return w
}

func (q *fiberWaitQueue) removeTicket(ticket uint32) *fiberWaiter {
	var prev *fiberWaiter
	for w := q.head; w != nil; w = w.next {
		if w.ticket == ticket {
			w.synctest.CheckWake()
			if prev == nil {
				q.head = w.next
			} else {
				prev.next = w.next
			}
			if q.tail == w {
				q.tail = prev
			}
			w.next = nil
			return w
		}
		prev = w
	}
	return nil
}

// semaQueues keys wait queues by semaphore address. wordmap hashes the key so
// no stale address bits survive deletion to pin the queue under the
// conservative collector.
var semaQueues wordmap.Map[uintptr, *fiberWaitQueue]

func semaQueue(addr *uint32) *fiberWaitQueue {
	key := uintptr(unsafe.Pointer(addr))
	q := semaQueues.Get(key)
	if q == nil {
		q = new(fiberWaitQueue)
		semaQueues.Set(key, q)
	}
	return q
}

// semaAcquire parks the current G until semaReleaseNative hands it a token.
// With no waiter queued, release increments the counter and a later acquire
// consumes it through the compare-and-swap fast path. Single-hart invariants:
// the push→Park sequence has no yield point, so a concurrent release can
// never interleave between enqueue and park, and interrupt handlers never
// touch the semaphore word (no ISR drives these addresses).
func semaAcquire(addr *uint32) {
	value := atomic.LoadUint32(addr)
	if value != 0 && atomic.CompareAndSwapUint32(addr, value, value-1) {
		return
	}
	w := acquireFiberWaiter(0)
	semaQueue(addr).push(w)
	w.synctest.Block()
	w.waiter.Park()
	releaseFiberWaiter(w)
}

// semaReleaseNative wakes one queued waiter, handing the token directly, or
// publishes it on the counter for the next fast-path acquirer. Same single-
// hart invariants as semaAcquire: the queue pop and the wake publish cannot
// interleave with another release.
//
// TODO: once the baremetal runtime grows a real nanotime, the mutex starvation
// path starts calling release with handoff=true; like the wasm backend, that
// must yield the hart (llruntime.Gosched) after Ready so the woken waiter
// runs before the releaser can reacquire. Today runtimeNano is constant zero,
// starvation never triggers, and dropping handoff stays correct.
func semaReleaseNative(addr *uint32) {
	key := uintptr(unsafe.Pointer(addr))
	if q := semaQueues.Get(key); q != nil {
		if w := q.pop(); w != nil {
			if q.head == nil {
				semaQueues.Delete(key)
			}
			w.synctest.Wake()
			w.waiter.Ready()
			return
		}
	}
	atomic.AddUint32(addr, 1)
}

// notifyList matches sync.notifyList size and layout; notifyListCheck
// verifies the agreement at init.
type notifyList struct {
	wait   uint32
	notify uint32
	lock   uintptr
	head   unsafe.Pointer
	tail   unsafe.Pointer
}

var notifyQueues wordmap.Map[uintptr, *fiberWaitQueue]

func notifyQueue(l *notifyList) *fiberWaitQueue {
	key := uintptr(unsafe.Pointer(l))
	q := notifyQueues.Get(key)
	if q == nil {
		q = new(fiberWaitQueue)
		notifyQueues.Set(key, q)
	}
	return q
}

func ticketLess(a, b uint32) bool {
	return int32(a-b) < 0
}

//go:linkname sync_runtime_Semacquire sync.runtime_Semacquire
func sync_runtime_Semacquire(addr *uint32) {
	semaAcquire(addr)
}

//go:linkname poll_runtime_Semacquire internal/poll.runtime_Semacquire
func poll_runtime_Semacquire(addr *uint32) {
	semaAcquire(addr)
}

//go:linkname sync_runtime_Semrelease sync.runtime_Semrelease
func sync_runtime_Semrelease(addr *uint32, handoff bool, _ int) {
	semaRelease(addr)
}

//go:linkname poll_runtime_Semrelease internal/poll.runtime_Semrelease
func poll_runtime_Semrelease(addr *uint32) {
	semaRelease(addr)
}

//go:linkname sync_runtime_SemacquireWaitGroup sync.runtime_SemacquireWaitGroup
func sync_runtime_SemacquireWaitGroup(addr *uint32, durable bool) {
	semaAcquireWaitGroup(addr, durable)
}

//go:linkname sync_runtime_SemacquireRWMutexR sync.runtime_SemacquireRWMutexR
func sync_runtime_SemacquireRWMutexR(addr *uint32, _ bool, _ int) {
	semaAcquire(addr)
}

//go:linkname sync_runtime_SemacquireRWMutex sync.runtime_SemacquireRWMutex
func sync_runtime_SemacquireRWMutex(addr *uint32, _ bool, _ int) {
	semaAcquire(addr)
}

//go:linkname sync_runtime_SemacquireMutex sync.runtime_SemacquireMutex
func sync_runtime_SemacquireMutex(addr *uint32, _ bool, _ int) {
	semaAcquire(addr)
}

func runtime_SemacquireMutex(addr *uint32, _ bool, _ int) {
	semaAcquire(addr)
}

func runtime_Semrelease(addr *uint32, _ bool, _ int) {
	semaRelease(addr)
}

func runtime_canSpin(int) bool { return false }
func runtime_doSpin()          {}

// runtime_nanotime feeds the mutex starvation threshold; the baremetal clock
// reads zero, which just keeps every waiter on the normal (non-starving)
// path, exactly as the pthread fallback does today.
func runtime_nanotime() int64 { return runtimeNano() }

//go:linkname sync_runtime_canSpin sync.runtime_canSpin
func sync_runtime_canSpin(i int) bool { return runtime_canSpin(i) }

//go:linkname sync_runtime_doSpin sync.runtime_doSpin
func sync_runtime_doSpin() { runtime_doSpin() }

//go:linkname sync_runtime_nanotime sync.runtime_nanotime
func sync_runtime_nanotime() int64 { return runtime_nanotime() }

//go:linkname internal_sync_runtime_canSpin internal/sync.runtime_canSpin
func internal_sync_runtime_canSpin(i int) bool { return runtime_canSpin(i) }

//go:linkname internal_sync_runtime_doSpin internal/sync.runtime_doSpin
func internal_sync_runtime_doSpin() { runtime_doSpin() }

//go:linkname internal_sync_runtime_nanotime internal/sync.runtime_nanotime
func internal_sync_runtime_nanotime() int64 { return runtime_nanotime() }

//go:linkname internal_sync_runtime_SemacquireMutex internal/sync.runtime_SemacquireMutex
func internal_sync_runtime_SemacquireMutex(addr *uint32, lifo bool, skipframes int) {
	runtime_SemacquireMutex(addr, lifo, skipframes)
}

//go:linkname internal_sync_runtime_Semrelease internal/sync.runtime_Semrelease
func internal_sync_runtime_Semrelease(addr *uint32, handoff bool, skipframes int) {
	runtime_Semrelease(addr, handoff, skipframes)
}

//go:linkname internal_sync_throw internal/sync.throw
func internal_sync_throw(s string) { throw(s) }

//go:linkname internal_sync_fatal internal/sync.fatal
func internal_sync_fatal(s string) { fatal(s) }

//go:linkname sync_runtime_notifyListAdd sync.runtime_notifyListAdd
func sync_runtime_notifyListAdd(l *notifyList) uint32 {
	return atomic.AddUint32(&l.wait, 1) - 1
}

//go:linkname sync_runtime_notifyListWait sync.runtime_notifyListWait
func sync_runtime_notifyListWait(l *notifyList, ticket uint32) {
	if ticketLess(ticket, atomic.LoadUint32(&l.notify)) {
		return
	}
	w := acquireFiberWaiter(ticket)
	w.synctest = llruntime.NewSynctestWaiter()
	notifyQueue(l).push(w)
	w.synctest.Block()
	w.waiter.Park()
	releaseFiberWaiter(w)
}

//go:linkname sync_runtime_notifyListNotifyAll sync.runtime_notifyListNotifyAll
func sync_runtime_notifyListNotifyAll(l *notifyList) {
	wait := atomic.LoadUint32(&l.wait)
	if atomic.LoadUint32(&l.notify) == wait {
		return
	}
	key := uintptr(unsafe.Pointer(l))
	q := notifyQueues.Get(key)
	q.checkWake()
	atomic.StoreUint32(&l.notify, wait)
	if q == nil {
		return
	}
	notifyQueues.Delete(key)
	for {
		w := q.pop()
		if w == nil {
			return
		}
		w.synctest.Wake()
		w.waiter.Ready()
	}
}

//go:linkname sync_runtime_notifyListNotifyOne sync.runtime_notifyListNotifyOne
func sync_runtime_notifyListNotifyOne(l *notifyList) {
	notify := atomic.LoadUint32(&l.notify)
	if notify == atomic.LoadUint32(&l.wait) {
		return
	}
	key := uintptr(unsafe.Pointer(l))
	q := notifyQueues.Get(key)
	if q == nil {
		atomic.StoreUint32(&l.notify, notify+1)
		return
	}
	w := q.removeTicket(notify)
	atomic.StoreUint32(&l.notify, notify+1)
	if w != nil {
		if q.head == nil {
			notifyQueues.Delete(key)
		}
		w.synctest.Wake()
		w.waiter.Ready()
	}
}

func (q *fiberWaitQueue) checkWake() {
	if q != nil {
		for w := q.head; w != nil; w = w.next {
			w.synctest.CheckWake()
		}
	}
}

//go:linkname sync_runtime_notifyListCheck sync.runtime_notifyListCheck
func sync_runtime_notifyListCheck(size uintptr) {
	if size != unsafe.Sizeof(notifyList{}) {
		panic("sync.notifyList size mismatch")
	}
}
