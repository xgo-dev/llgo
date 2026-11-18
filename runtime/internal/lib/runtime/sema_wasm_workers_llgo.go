//go:build llgo && js && wasm && llgo.wasm.workers

package runtime

import (
	"sync/atomic"
	"unsafe"

	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
	"github.com/xgo-dev/llgo/runtime/internal/wasmsync"
)

// wasmWaitQueuesLock protects both queue tables and their shared waiter free
// list. Separate table locks would still race through wasmWaiterFree.
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

func semaAcquire(addr *uint32, lifo bool) { semaAcquireInternal(addr, lifo, false) }

func semaAcquireInternal(addr *uint32, lifo, durable bool) {
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
	if durable {
		w.synctest = llruntime.NewSynctestWaiter()
	}
	semaQueue(addr).push(w, lifo)
	w.synctest.Block()
	wasmWaitQueuesLock.Unlock()
	w.waiter.Park()
	wasmWaitQueuesLock.Lock()
	releaseWasmWaiter(w)
	wasmWaitQueuesLock.Unlock()
}

func semaRelease(addr *uint32, handoff bool) {
	key := uintptr(unsafe.Pointer(addr))
	w := func() *wasmWaiter {
		wasmWaitQueuesLock.Lock()
		defer wasmWaitQueuesLock.Unlock()
		if q := semaQueues.Get(key); q != nil {
			if w := q.pop(); w != nil {
				if q.head == nil {
					semaQueues.Delete(key)
				}
				w.synctest.Wake()
				return w
			}
		}
		atomic.AddUint32(addr, 1)
		return nil
	}()
	if w != nil {
		w.waiter.Ready()
		if handoff {
			llruntime.Gosched()
		}
	}
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
	w.synctest = llruntime.NewSynctestWaiter()
	notifyQueue(l).push(w, false)
	w.synctest.Block()
	wasmWaitQueuesLock.Unlock()
	w.waiter.Park()
	wasmWaitQueuesLock.Lock()
	releaseWasmWaiter(w)
	wasmWaitQueuesLock.Unlock()
}

//go:linkname sync_runtime_notifyListNotifyAll sync.runtime_notifyListNotifyAll
func sync_runtime_notifyListNotifyAll(l *notifyList) {
	q := func() *wasmWaitQueue {
		wasmWaitQueuesLock.Lock()
		defer wasmWaitQueuesLock.Unlock()
		wait := atomic.LoadUint32(&l.wait)
		if atomic.LoadUint32(&l.notify) == wait {
			return nil
		}
		key := uintptr(unsafe.Pointer(l))
		q := notifyQueues.Get(key)
		q.checkWake()
		atomic.StoreUint32(&l.notify, wait)
		notifyQueues.Delete(key)
		return q
	}()
	if q == nil {
		return
	}
	for w := q.pop(); w != nil; w = q.pop() {
		w.synctest.Wake()
		w.waiter.Ready()
	}
}

//go:linkname sync_runtime_notifyListNotifyOne sync.runtime_notifyListNotifyOne
func sync_runtime_notifyListNotifyOne(l *notifyList) {
	w := func() *wasmWaiter {
		wasmWaitQueuesLock.Lock()
		defer wasmWaitQueuesLock.Unlock()
		notify := atomic.LoadUint32(&l.notify)
		if notify == atomic.LoadUint32(&l.wait) {
			return nil
		}
		key := uintptr(unsafe.Pointer(l))
		q := notifyQueues.Get(key)
		var w *wasmWaiter
		if q != nil {
			w = q.removeTicket(notify)
			if q.head == nil {
				notifyQueues.Delete(key)
			}
		}
		atomic.StoreUint32(&l.notify, notify+1)
		if w != nil {
			w.synctest.Wake()
		}
		return w
	}()
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
