//go:build llgo && wasm && !(wasip1 && llgo.wasi_threads) && !llgo.wasm_workers

package runtime

import (
	"unsafe"

	latomic "github.com/goplus/llgo/runtime/internal/lib/sync/atomic"
	llruntime "github.com/goplus/llgo/runtime/internal/runtime"
)

func semaAcquire(addr *uint32, lifo bool) {
	value := latomic.LoadUint32(addr)
	if value != 0 && latomic.CompareAndSwapUint32(addr, value, value-1) {
		return
	}
	w := &wasmWaiter{waiter: llruntime.CurrentSchedulerWaiter()}
	semaQueue(addr).push(w, lifo)
	w.waiter.Park()
}

func semaRelease(addr *uint32, handoff bool) {
	key := uintptr(unsafe.Pointer(addr))
	if q := semaQueues[key]; q != nil {
		if w := q.pop(); w != nil {
			if q.head == nil {
				delete(semaQueues, key)
			}
			w.waiter.Ready()
			if handoff {
				llruntime.Gosched()
			}
			return
		}
	}
	latomic.AddUint32(addr, 1)
}

//go:linkname sync_runtime_notifyListWait sync.runtime_notifyListWait
func sync_runtime_notifyListWait(l *notifyList, ticket uint32) {
	if ticketLess(ticket, latomic.LoadUint32(&l.notify)) {
		return
	}
	w := &wasmWaiter{
		waiter: llruntime.CurrentSchedulerWaiter(),
		ticket: ticket,
	}
	notifyQueue(l).push(w, false)
	w.waiter.Park()
}

//go:linkname sync_runtime_notifyListNotifyAll sync.runtime_notifyListNotifyAll
func sync_runtime_notifyListNotifyAll(l *notifyList) {
	wait := latomic.LoadUint32(&l.wait)
	if latomic.LoadUint32(&l.notify) == wait {
		return
	}
	latomic.StoreUint32(&l.notify, wait)
	key := uintptr(unsafe.Pointer(l))
	q := notifyQueues[key]
	if q == nil {
		return
	}
	delete(notifyQueues, key)
	for {
		w := q.pop()
		if w == nil {
			return
		}
		w.waiter.Ready()
	}
}

//go:linkname sync_runtime_notifyListNotifyOne sync.runtime_notifyListNotifyOne
func sync_runtime_notifyListNotifyOne(l *notifyList) {
	notify := latomic.LoadUint32(&l.notify)
	if notify == latomic.LoadUint32(&l.wait) {
		return
	}
	latomic.StoreUint32(&l.notify, notify+1)
	key := uintptr(unsafe.Pointer(l))
	q := notifyQueues[key]
	if q == nil {
		return
	}
	if w := q.removeTicket(notify); w != nil {
		if q.head == nil {
			delete(notifyQueues, key)
		}
		w.waiter.Ready()
	}
}

//go:linkname sync_runtime_procPin sync.runtime_procPin
func sync_runtime_procPin() int {
	return 0
}

//go:linkname sync_runtime_procUnpin sync.runtime_procUnpin
func sync_runtime_procUnpin() {}

//go:linkname atomic_runtime_procPin sync/atomic.runtime_procPin
func atomic_runtime_procPin() int {
	return 0
}

//go:linkname atomic_runtime_procUnpin sync/atomic.runtime_procUnpin
func atomic_runtime_procUnpin() {}
