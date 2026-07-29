//go:build llgo && js && wasm && llgo.wasm_workers

package runtime

import (
	"unsafe"

	psync "github.com/goplus/llgo/runtime/internal/clite/pthread/sync"
	latomic "github.com/goplus/llgo/runtime/internal/lib/sync/atomic"
	llruntime "github.com/goplus/llgo/runtime/internal/runtime"
)

var semaQueuesLock = newWasmSemaMutex()
var notifyQueuesLock = newWasmSemaMutex()

type wasmSemaMutex struct {
	mutex psync.Mutex
}

func newWasmSemaMutex() wasmSemaMutex {
	var result wasmSemaMutex
	if result.mutex.Init(nil) != 0 {
		panic("runtime: failed to initialize WebAssembly semaphore mutex")
	}
	return result
}

func (m *wasmSemaMutex) Lock() {
	m.mutex.Lock()
}

func (m *wasmSemaMutex) Unlock() {
	m.mutex.Unlock()
}

func semaAcquire(addr *uint32, lifo bool) {
	value := latomic.LoadUint32(addr)
	if value != 0 && latomic.CompareAndSwapUint32(addr, value, value-1) {
		return
	}
	semaQueuesLock.Lock()
	value = latomic.LoadUint32(addr)
	if value != 0 && latomic.CompareAndSwapUint32(addr, value, value-1) {
		semaQueuesLock.Unlock()
		return
	}
	w := &wasmWaiter{waiter: llruntime.CurrentSchedulerWaiter()}
	semaQueue(addr).push(w, lifo)
	semaQueuesLock.Unlock()
	w.waiter.Park()
}

func semaRelease(addr *uint32, handoff bool) {
	key := uintptr(unsafe.Pointer(addr))
	semaQueuesLock.Lock()
	if q := semaQueues[key]; q != nil {
		if w := q.pop(); w != nil {
			if q.head == nil {
				delete(semaQueues, key)
			}
			semaQueuesLock.Unlock()
			w.waiter.Ready()
			if handoff {
				llruntime.Gosched()
			}
			return
		}
	}
	latomic.AddUint32(addr, 1)
	semaQueuesLock.Unlock()
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
	notifyQueuesLock.Lock()
	if ticketLess(ticket, latomic.LoadUint32(&l.notify)) {
		notifyQueuesLock.Unlock()
		return
	}
	notifyQueue(l).push(w, false)
	notifyQueuesLock.Unlock()
	w.waiter.Park()
}

//go:linkname sync_runtime_notifyListNotifyAll sync.runtime_notifyListNotifyAll
func sync_runtime_notifyListNotifyAll(l *notifyList) {
	notifyQueuesLock.Lock()
	wait := latomic.LoadUint32(&l.wait)
	if latomic.LoadUint32(&l.notify) == wait {
		notifyQueuesLock.Unlock()
		return
	}
	latomic.StoreUint32(&l.notify, wait)
	key := uintptr(unsafe.Pointer(l))
	q := notifyQueues[key]
	if q == nil {
		notifyQueuesLock.Unlock()
		return
	}
	delete(notifyQueues, key)
	notifyQueuesLock.Unlock()
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
	notifyQueuesLock.Lock()
	notify := latomic.LoadUint32(&l.notify)
	if notify == latomic.LoadUint32(&l.wait) {
		notifyQueuesLock.Unlock()
		return
	}
	latomic.StoreUint32(&l.notify, notify+1)
	key := uintptr(unsafe.Pointer(l))
	q := notifyQueues[key]
	if q == nil {
		notifyQueuesLock.Unlock()
		return
	}
	w := q.removeTicket(notify)
	if q.head == nil {
		delete(notifyQueues, key)
	}
	notifyQueuesLock.Unlock()
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

//go:linkname atomic_runtime_procPin sync/atomic.runtime_procPin
func atomic_runtime_procPin() int {
	return llruntime.SchedulerProcID()
}

//go:linkname atomic_runtime_procUnpin sync/atomic.runtime_procUnpin
func atomic_runtime_procUnpin() {}
