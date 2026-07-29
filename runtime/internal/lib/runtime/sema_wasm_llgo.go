//go:build llgo && wasm && !(wasip1 && llgo.wasi_threads)

package runtime

import (
	"unsafe"

	latomic "github.com/goplus/llgo/runtime/internal/lib/sync/atomic"
	llruntime "github.com/goplus/llgo/runtime/internal/runtime"
)

type wasmWaiter struct {
	next   *wasmWaiter
	waiter llruntime.SchedulerWaiter
	ticket uint32
}

type wasmWaitQueue struct {
	head *wasmWaiter
	tail *wasmWaiter
}

func (q *wasmWaitQueue) push(w *wasmWaiter, lifo bool) {
	if lifo {
		w.next = q.head
		q.head = w
		if q.tail == nil {
			q.tail = w
		}
		return
	}
	if q.tail == nil {
		q.head = w
	} else {
		q.tail.next = w
	}
	q.tail = w
}

func (q *wasmWaitQueue) pop() *wasmWaiter {
	w := q.head
	if w == nil {
		return nil
	}
	q.head = w.next
	if q.head == nil {
		q.tail = nil
	}
	w.next = nil
	return w
}

func (q *wasmWaitQueue) removeTicket(ticket uint32) *wasmWaiter {
	var prev *wasmWaiter
	for w := q.head; w != nil; w = w.next {
		if w.ticket == ticket {
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

var semaQueues map[uintptr]*wasmWaitQueue

func semaQueue(addr *uint32) *wasmWaitQueue {
	if semaQueues == nil {
		semaQueues = make(map[uintptr]*wasmWaitQueue)
	}
	key := uintptr(unsafe.Pointer(addr))
	q := semaQueues[key]
	if q == nil {
		q = new(wasmWaitQueue)
		semaQueues[key] = q
	}
	return q
}

//go:linkname sync_runtime_Semacquire sync.runtime_Semacquire
func sync_runtime_Semacquire(addr *uint32) {
	semaAcquire(addr, false)
}

//go:linkname poll_runtime_Semacquire internal/poll.runtime_Semacquire
func poll_runtime_Semacquire(addr *uint32) {
	semaAcquire(addr, false)
}

//go:linkname sync_runtime_Semrelease sync.runtime_Semrelease
func sync_runtime_Semrelease(addr *uint32, handoff bool, _ int) {
	semaRelease(addr, handoff)
}

//go:linkname sync_runtime_SemacquireRWMutexR sync.runtime_SemacquireRWMutexR
func sync_runtime_SemacquireRWMutexR(addr *uint32, lifo bool, _ int) {
	semaAcquire(addr, lifo)
}

//go:linkname sync_runtime_SemacquireRWMutex sync.runtime_SemacquireRWMutex
func sync_runtime_SemacquireRWMutex(addr *uint32, lifo bool, _ int) {
	semaAcquire(addr, lifo)
}

func syncWaitGroupAcquire(addr *uint32) {
	semaAcquire(addr, false)
}

func runtime_SemacquireMutex(addr *uint32, lifo bool, _ int) {
	semaAcquire(addr, lifo)
}

//go:linkname sync_runtime_SemacquireMutex sync.runtime_SemacquireMutex
func sync_runtime_SemacquireMutex(addr *uint32, lifo bool, skipframes int) {
	runtime_SemacquireMutex(addr, lifo, skipframes)
}

func runtime_Semrelease(addr *uint32, handoff bool, _ int) {
	semaRelease(addr, handoff)
}

//go:linkname poll_runtime_Semrelease internal/poll.runtime_Semrelease
func poll_runtime_Semrelease(addr *uint32) {
	semaRelease(addr, false)
}

func runtime_canSpin(int) bool { return false }
func runtime_doSpin()          {}
func runtime_nanotime() int64  { return runtimeNano() }

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

type notifyList struct {
	wait   uint32
	notify uint32
	lock   uintptr
	head   unsafe.Pointer
	tail   unsafe.Pointer
}

var notifyQueues map[uintptr]*wasmWaitQueue

func notifyQueue(l *notifyList) *wasmWaitQueue {
	if notifyQueues == nil {
		notifyQueues = make(map[uintptr]*wasmWaitQueue)
	}
	key := uintptr(unsafe.Pointer(l))
	q := notifyQueues[key]
	if q == nil {
		q = new(wasmWaitQueue)
		notifyQueues[key] = q
	}
	return q
}

func ticketLess(a, b uint32) bool {
	return int32(a-b) < 0
}

//go:linkname sync_runtime_notifyListAdd sync.runtime_notifyListAdd
func sync_runtime_notifyListAdd(l *notifyList) uint32 {
	return latomic.AddUint32(&l.wait, 1) - 1
}

//go:linkname sync_runtime_notifyListCheck sync.runtime_notifyListCheck
func sync_runtime_notifyListCheck(size uintptr) {
	if size != unsafe.Sizeof(notifyList{}) {
		panic("sync.notifyList size mismatch")
	}
}

var poolCleanup func()

//go:linkname sync_runtime_registerPoolCleanup sync.runtime_registerPoolCleanup
func sync_runtime_registerPoolCleanup(cleanup func()) {
	poolCleanup = cleanup
}
