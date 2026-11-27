//go:build darwin || linux || windows || (llgo && wasip1 && wasm && llgo.wasi_threads)

package runtime

import (
	"unsafe"

	latomic "sync/atomic"

	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
)

// Minimal semaphore + notify list support for stdlib sync on hosted targets.

// sync_runtime_Semacquire should be an internal detail, but is linknamed.
//
//go:linkname sync_runtime_Semacquire sync.runtime_Semacquire
func sync_runtime_Semacquire(addr *uint32) {
	semaAcquire(addr)
}

//go:linkname poll_runtime_Semacquire internal/poll.runtime_Semacquire
func poll_runtime_Semacquire(addr *uint32) {
	semaAcquire(addr)
}

//go:linkname sync_runtime_Semrelease sync.runtime_Semrelease
func sync_runtime_Semrelease(addr *uint32, _ bool, _ int) {
	semaRelease(addr)
}

//go:linkname sync_runtime_SemacquireRWMutexR sync.runtime_SemacquireRWMutexR
func sync_runtime_SemacquireRWMutexR(addr *uint32, _ bool, _ int) {
	semaAcquire(addr)
}

//go:linkname sync_runtime_SemacquireRWMutex sync.runtime_SemacquireRWMutex
func sync_runtime_SemacquireRWMutex(addr *uint32, _ bool, _ int) {
	semaAcquire(addr)
}

//go:linkname sync_runtime_SemacquireWaitGroup sync.runtime_SemacquireWaitGroup
func sync_runtime_SemacquireWaitGroup(addr *uint32, durable bool) {
	semaAcquireWaitGroup(addr, durable)
}

// runtime_SemacquireMutex is used by internal/sync via linkname.
func runtime_SemacquireMutex(addr *uint32, _ bool, _ int) {
	semaAcquire(addr)
}

// sync_runtime_SemacquireMutex is used by older stdlib sync implementations.
//
//go:linkname sync_runtime_SemacquireMutex sync.runtime_SemacquireMutex
func sync_runtime_SemacquireMutex(addr *uint32, lifo bool, skipframes int) {
	runtime_SemacquireMutex(addr, lifo, skipframes)
}

// runtime_Semrelease is used by internal/sync via linkname.
func runtime_Semrelease(addr *uint32, _ bool, _ int) {
	semaRelease(addr)
}

//go:linkname poll_runtime_Semrelease internal/poll.runtime_Semrelease
func poll_runtime_Semrelease(addr *uint32) {
	semaRelease(addr)
}

var semaSpinCPUs uint32

func semaCanSpinMultiCPU() bool {
	n := latomic.LoadUint32(&semaSpinCPUs)
	if n == 0 {
		c := rt.CPUCount()
		if c < 1 {
			c = 1
		}
		latomic.StoreUint32(&semaSpinCPUs, uint32(c))
		n = uint32(c)
	}
	return n > 1
}

// runtime_canSpin/runtime_doSpin/runtime_nanotime are used by internal/sync.
// Match Go's bounded 4-round short spin for non-starving mutexes on multi-core
// hosts so short critical sections avoid kernel sleep/wake round-trips.
func runtime_canSpin(i int) bool {
	return i < 4 && semaCanSpinMultiCPU()
}

func runtime_doSpin() {
	rt.Procyield(10)
}

func runtime_nanotime() int64 { return runtimeNano() }

// sync_runtime_canSpin/sync_runtime_doSpin/sync_runtime_nanotime are used by
// older stdlib sync implementations.
//
//go:linkname sync_runtime_canSpin sync.runtime_canSpin
func sync_runtime_canSpin(i int) bool { return runtime_canSpin(i) }

//go:linkname sync_runtime_doSpin sync.runtime_doSpin
func sync_runtime_doSpin() { runtime_doSpin() }

//go:linkname sync_runtime_nanotime sync.runtime_nanotime
func sync_runtime_nanotime() int64 { return runtime_nanotime() }

// internal/sync uses these runtime hooks on Go 1.24+.
//
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

// notifyList matches sync.notifyList size/layout.
type notifyList struct {
	wait   uint32
	notify uint32
	lock   uintptr
	head   unsafe.Pointer
	tail   unsafe.Pointer
}

type notifyState struct {
	// refs counts calls using this state, including parked waiters. notifyMu
	// protects it so an idle list cannot retain its address and native resources.
	refs    uint32
	waiters map[uint32]*rt.SynctestWaiter
	mu      psync.Mutex
	cond    psync.Cond
}

var notifyOnce psync.Once
var notifyMu psync.Mutex

// Pointer keys are cleared by map deletion. A uintptr key can leave stale
// address bits in an empty bucket, retaining the Cond under conservative GC.
var notifyMap map[*notifyList]*notifyState

func initNotifyMap() {
	notifyMu.Init(nil)
	notifyMap = make(map[*notifyList]*notifyState)
}

// syncNotifyStateCountForTesting observes the side table without depending on
// conservative GC's treatment of stale address bits, especially on 32-bit hosts.
func syncNotifyStateCountForTesting() int {
	notifyOnce.Do(initNotifyMap)
	notifyMu.Lock()
	n := len(notifyMap)
	notifyMu.Unlock()
	return n
}

func getNotifyState(l *notifyList) *notifyState {
	notifyOnce.Do(initNotifyMap)
	notifyMu.Lock()
	st := notifyMap[l]
	if st == nil {
		st = &notifyState{}
		st.mu.Init(nil)
		st.cond.Init(nil)
		notifyMap[l] = st
	}
	st.refs++
	notifyMu.Unlock()
	return st
}

func releaseNotifyState(l *notifyList, st *notifyState) {
	notifyMu.Lock()
	st.refs--
	idle := st.refs == 0
	if idle {
		delete(notifyMap, l)
	}
	notifyMu.Unlock()
	if idle {
		st.cond.Destroy()
		st.mu.Destroy()
	}
}

//go:linkname sync_runtime_notifyListAdd sync.runtime_notifyListAdd
func sync_runtime_notifyListAdd(l *notifyList) uint32 {
	return latomic.AddUint32(&l.wait, 1) - 1
}

//go:linkname sync_runtime_notifyListWait sync.runtime_notifyListWait
func sync_runtime_notifyListWait(l *notifyList, t uint32) {
	if int32(t-latomic.LoadUint32(&l.notify)) < 0 {
		return
	}
	st := getNotifyState(l)
	defer releaseNotifyState(l, st)
	st.mu.Lock()
	var waiter *rt.SynctestWaiter
	if int32(t-latomic.LoadUint32(&l.notify)) >= 0 {
		waiter = rt.NewSynctestWaiter()
		if waiter != nil {
			if st.waiters == nil {
				st.waiters = make(map[uint32]*rt.SynctestWaiter)
			}
			st.waiters[t] = waiter
		}
	}
	for int32(t-latomic.LoadUint32(&l.notify)) >= 0 {
		waiter.Block()
		st.cond.Wait(&st.mu)
	}
	delete(st.waiters, t)
	st.mu.Unlock()
}

//go:linkname sync_runtime_notifyListNotifyAll sync.runtime_notifyListNotifyAll
func sync_runtime_notifyListNotifyAll(l *notifyList) {
	if latomic.LoadUint32(&l.wait) == latomic.LoadUint32(&l.notify) {
		return
	}
	st := getNotifyState(l)
	defer releaseNotifyState(l, st)
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, w := range st.waiters {
		w.CheckWake()
	}
	for _, w := range st.waiters {
		w.Wake()
	}
	latomic.StoreUint32(&l.notify, latomic.LoadUint32(&l.wait))
	st.cond.Broadcast()
}

//go:linkname sync_runtime_notifyListNotifyOne sync.runtime_notifyListNotifyOne
func sync_runtime_notifyListNotifyOne(l *notifyList) {
	if latomic.LoadUint32(&l.wait) == latomic.LoadUint32(&l.notify) {
		return
	}
	st := getNotifyState(l)
	defer releaseNotifyState(l, st)
	st.mu.Lock()
	defer st.mu.Unlock()
	if n := latomic.LoadUint32(&l.notify); n != latomic.LoadUint32(&l.wait) {
		st.waiters[n].Wake()
		latomic.StoreUint32(&l.notify, n+1)
		// The shared pthread condition cannot target a ticket. Signal could
		// wake a different waiter, so broadcast and recheck tickets in Wait.
		st.cond.Broadcast()
	}
}

//go:linkname sync_runtime_notifyListCheck sync.runtime_notifyListCheck
func sync_runtime_notifyListCheck(size uintptr) {
	if size != unsafe.Sizeof(notifyList{}) {
		panic("sync.notifyList size mismatch")
	}
}
