//go:build darwin

package runtime

import (
	"unsafe"

	latomic "sync/atomic"

	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
)

// On macOS 14.4+, use os_sync_wait_on_address / os_sync_wake_by_address_any
// directly so sync.Mutex/RWMutex/WaitGroup semaphores are lock-free and never
// allocate or retain per-address entries in a global map. Older macOS versions
// fall back to the condvar map below.
var darwinSemaBackend uint32 // 0=unknown, 1=cond, 2=address

func darwinUseAddressSema() bool {
	b := latomic.LoadUint32(&darwinSemaBackend)
	if b != 0 {
		return b == 2
	}
	if psync.HasWaitUint32() {
		latomic.StoreUint32(&darwinSemaBackend, 2)
		return true
	}
	latomic.StoreUint32(&darwinSemaBackend, 1)
	return false
}

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
	if darwinUseAddressSema() {
		for {
			value := latomic.LoadUint32(addr)
			if value != 0 && latomic.CompareAndSwapUint32(addr, value, value-1) {
				return
			}
			psync.WaitUint32(addr, 0)
		}
	}
	for {
		v := latomic.LoadUint32(addr)
		if v != 0 && latomic.CompareAndSwapUint32(addr, v, v-1) {
			return
		}
		st := getSemaState(addr)
		st.mu.Lock()
		for {
			v = latomic.LoadUint32(addr)
			if v != 0 {
				if latomic.CompareAndSwapUint32(addr, v, v-1) {
					st.mu.Unlock()
					return
				}
				continue
			}
			st.waiters++
			st.cond.Wait(&st.mu)
			st.waiters--
		}
	}
}

func semaReleaseNative(addr *uint32) {
	if darwinUseAddressSema() {
		latomic.AddUint32(addr, 1)
		psync.WakeUint32(addr)
		return
	}
	latomic.AddUint32(addr, 1)
	st := getSemaState(addr)
	st.mu.Lock()
	if st.waiters != 0 {
		st.cond.Signal()
	}
	st.mu.Unlock()
}
