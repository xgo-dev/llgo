//go:build llgo && js && wasm && llgo.wasm.workers

package runtime

import (
	"sync/atomic"
	"unsafe"

	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
	"github.com/xgo-dev/llgo/runtime/internal/wasmsync"
)

// wasmWaitQueuesLock protects both queue maps and their shared waiter free
// list. Separate map locks would still race through wasmWaiterFree.
var wasmWaitQueuesLock wasmSemaMutex

type wasmSemaMutex struct {
	mutex wasmsync.Mutex
}

func (m *wasmSemaMutex) Lock() {
	m.mutex.Lock(llruntime.CooperativeSafepoint)
}

func (m *wasmSemaMutex) Unlock() {
	m.mutex.Unlock()
}

func semaAcquire(addr *uint32, lifo bool) {
	value := atomic.LoadUint32(addr)
	if value != 0 && atomic.CompareAndSwapUint32(addr, value, value-1) {
		return
	}
	wasmWaitQueuesLock.Lock()
	value = atomic.LoadUint32(addr)
	if value != 0 && atomic.CompareAndSwapUint32(addr, value, value-1) {
		wasmWaitQueuesLock.Unlock()
		return
	}
	w := acquireWasmWaiter(0)
	semaQueue(addr).push(w, lifo)
	wasmWaitQueuesLock.Unlock()
	w.waiter.Park()
	wasmWaitQueuesLock.Lock()
	releaseWasmWaiter(w)
	wasmWaitQueuesLock.Unlock()
}

func semaRelease(addr *uint32, handoff bool) {
	key := uintptr(unsafe.Pointer(addr))
	wasmWaitQueuesLock.Lock()
	if q := semaQueues[key]; q != nil {
		if w := q.pop(); w != nil {
			if q.head == nil {
				delete(semaQueues, key)
			}
			wasmWaitQueuesLock.Unlock()
			w.waiter.Ready()
			if handoff {
				llruntime.Gosched()
			}
			return
		}
	}
	atomic.AddUint32(addr, 1)
	wasmWaitQueuesLock.Unlock()
}

//go:linkname sync_runtime_notifyListWait sync.runtime_notifyListWait
func sync_runtime_notifyListWait(l *notifyList, ticket uint32) {
	if ticketLess(ticket, atomic.LoadUint32(&l.notify)) {
		return
	}
	wasmWaitQueuesLock.Lock()
	if ticketLess(ticket, atomic.LoadUint32(&l.notify)) {
		wasmWaitQueuesLock.Unlock()
		return
	}
	w := acquireWasmWaiter(ticket)
	notifyQueue(l).push(w, false)
	wasmWaitQueuesLock.Unlock()
	w.waiter.Park()
	wasmWaitQueuesLock.Lock()
	releaseWasmWaiter(w)
	wasmWaitQueuesLock.Unlock()
}

//go:linkname sync_runtime_notifyListNotifyAll sync.runtime_notifyListNotifyAll
func sync_runtime_notifyListNotifyAll(l *notifyList) {
	wasmWaitQueuesLock.Lock()
	wait := atomic.LoadUint32(&l.wait)
	if atomic.LoadUint32(&l.notify) == wait {
		wasmWaitQueuesLock.Unlock()
		return
	}
	atomic.StoreUint32(&l.notify, wait)
	key := uintptr(unsafe.Pointer(l))
	q := notifyQueues[key]
	if q == nil {
		wasmWaitQueuesLock.Unlock()
		return
	}
	delete(notifyQueues, key)
	wasmWaitQueuesLock.Unlock()
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
	wasmWaitQueuesLock.Lock()
	notify := atomic.LoadUint32(&l.notify)
	if notify == atomic.LoadUint32(&l.wait) {
		wasmWaitQueuesLock.Unlock()
		return
	}
	atomic.StoreUint32(&l.notify, notify+1)
	key := uintptr(unsafe.Pointer(l))
	q := notifyQueues[key]
	if q == nil {
		wasmWaitQueuesLock.Unlock()
		return
	}
	w := q.removeTicket(notify)
	if q.head == nil {
		delete(notifyQueues, key)
	}
	wasmWaitQueuesLock.Unlock()
	if w != nil {
		w.waiter.Ready()
	}
}

//go:linkname sync_runtime_procPin sync.runtime_procPin
func sync_runtime_procPin() int {
	return llruntime.SchedulerProcID()
}

//go:linkname sync_runtime_procUnpin sync.runtime_procUnpin
func sync_runtime_procUnpin() {}
