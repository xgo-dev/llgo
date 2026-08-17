//go:build darwin || linux

package runtime

import (
	"unsafe"

	latomic "sync/atomic"

	psync "github.com/xgo-dev/llgo/runtime/internal/clite/pthread/sync"
)

// Minimal semaphore + notify list support for stdlib sync on llgo/darwin.

type semaState struct {
	mu      psync.Mutex
	cond    psync.Cond
	waiters uint32
}

var semaOnce psync.Once
var semaMu psync.Mutex
var semaMap map[uintptr]*semaState

func initSemaMap() {
	semaMu.Init(nil)
	semaMap = make(map[uintptr]*semaState)
}

func getSemaState(addr *uint32) *semaState {
	semaOnce.Do(initSemaMap)
	key := uintptr(unsafe.Pointer(addr))
	semaMu.Lock()
	st := semaMap[key]
	if st == nil {
		st = &semaState{}
		st.mu.Init(nil)
		st.cond.Init(nil)
		semaMap[key] = st
	}
	semaMu.Unlock()
	return st
}

func semaAcquire(addr *uint32) {
	for {
		v := latomic.LoadUint32(addr)
		if v != 0 && latomic.CompareAndSwapUint32(addr, v, v-1) {
			return
		}
		st := getSemaState(addr)
		st.mu.Lock()
		for {
			v = latomic.LoadUint32(addr)
			if v != 0 && latomic.CompareAndSwapUint32(addr, v, v-1) {
				st.mu.Unlock()
				return
			}
			st.waiters++
			st.cond.Wait(&st.mu)
			st.waiters--
		}
	}
}

func semaRelease(addr *uint32) {
	latomic.AddUint32(addr, 1)
	st := getSemaState(addr)
	st.mu.Lock()
	if st.waiters != 0 {
		st.cond.Signal()
	}
	st.mu.Unlock()
}

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
func sync_runtime_SemacquireWaitGroup(addr *uint32, _ bool) {
	semaAcquire(addr)
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

// runtime_canSpin/runtime_doSpin/runtime_nanotime are used by internal/sync.
func runtime_canSpin(_ int) bool { return false }
func runtime_doSpin()            {}
func runtime_nanotime() int64    { return runtimeNano() }

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
	mu   psync.Mutex
	cond psync.Cond
}

var notifyOnce psync.Once
var notifyMu psync.Mutex
var notifyMap map[uintptr]*notifyState

func initNotifyMap() {
	notifyMu.Init(nil)
	notifyMap = make(map[uintptr]*notifyState)
}

func getNotifyState(l *notifyList) *notifyState {
	notifyOnce.Do(initNotifyMap)
	key := uintptr(unsafe.Pointer(l))
	notifyMu.Lock()
	st := notifyMap[key]
	if st == nil {
		st = &notifyState{}
		st.mu.Init(nil)
		st.cond.Init(nil)
		notifyMap[key] = st
	}
	notifyMu.Unlock()
	return st
}

//go:linkname sync_runtime_notifyListAdd sync.runtime_notifyListAdd
func sync_runtime_notifyListAdd(l *notifyList) uint32 {
	return latomic.AddUint32(&l.wait, 1) - 1
}

//go:linkname sync_runtime_notifyListWait sync.runtime_notifyListWait
func sync_runtime_notifyListWait(l *notifyList, t uint32) {
	st := getNotifyState(l)
	st.mu.Lock()
	for latomic.LoadUint32(&l.notify) == t {
		st.cond.Wait(&st.mu)
	}
	st.mu.Unlock()
}

//go:linkname sync_runtime_notifyListNotifyAll sync.runtime_notifyListNotifyAll
func sync_runtime_notifyListNotifyAll(l *notifyList) {
	st := getNotifyState(l)
	st.mu.Lock()
	latomic.StoreUint32(&l.notify, latomic.LoadUint32(&l.wait))
	st.cond.Broadcast()
	st.mu.Unlock()
}

//go:linkname sync_runtime_notifyListNotifyOne sync.runtime_notifyListNotifyOne
func sync_runtime_notifyListNotifyOne(l *notifyList) {
	st := getNotifyState(l)
	st.mu.Lock()
	if latomic.LoadUint32(&l.notify) != latomic.LoadUint32(&l.wait) {
		latomic.AddUint32(&l.notify, 1)
		st.cond.Signal()
	}
	st.mu.Unlock()
}

//go:linkname sync_runtime_notifyListCheck sync.runtime_notifyListCheck
func sync_runtime_notifyListCheck(size uintptr) {
	if size != unsafe.Sizeof(notifyList{}) {
		panic("sync.notifyList size mismatch")
	}
}
