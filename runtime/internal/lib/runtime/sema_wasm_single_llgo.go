//go:build llgo && wasm && !(wasip1 && llgo.wasi_threads) && !llgo.wasm.workers

package runtime

import (
	"sync/atomic"
	"unsafe"

	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

func semaAcquire(addr *uint32, lifo bool) {
	value := atomic.LoadUint32(addr)
	if value != 0 && atomic.CompareAndSwapUint32(addr, value, value-1) {
		return
	}
	w := acquireWasmWaiter(0)
	semaQueue(addr).push(w, lifo)
	w.waiter.Park()
	releaseWasmWaiter(w)
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
	atomic.AddUint32(addr, 1)
}

//go:linkname sync_runtime_notifyListWait sync.runtime_notifyListWait
func sync_runtime_notifyListWait(l *notifyList, ticket uint32) {
	if ticketLess(ticket, atomic.LoadUint32(&l.notify)) {
		return
	}
	w := acquireWasmWaiter(ticket)
	notifyQueue(l).push(w, false)
	w.waiter.Park()
	releaseWasmWaiter(w)
}

//go:linkname sync_runtime_notifyListNotifyAll sync.runtime_notifyListNotifyAll
func sync_runtime_notifyListNotifyAll(l *notifyList) {
	wait := atomic.LoadUint32(&l.wait)
	if atomic.LoadUint32(&l.notify) == wait {
		return
	}
	atomic.StoreUint32(&l.notify, wait)
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
	notify := atomic.LoadUint32(&l.notify)
	if notify == atomic.LoadUint32(&l.wait) {
		return
	}
	atomic.StoreUint32(&l.notify, notify+1)
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
