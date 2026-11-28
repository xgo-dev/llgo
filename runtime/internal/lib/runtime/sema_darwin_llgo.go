//go:build darwin

package runtime

import (
	latomic "sync/atomic"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
)

// On macOS 14.4+, use os_sync_wait_on_address / os_sync_wake_by_address_any
// directly so sync.Mutex/RWMutex/WaitGroup semaphores are lock-free and never
// allocate or retain per-address entries in a global map. Older macOS versions
// fall back to a refcounted condvar side table keyed by *uint32.
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
	// refs counts active acquirers/releasers using this state, including parked
	// waiters. Protected by semaMu so idle entries and their native mutex/cond
	// objects are freed instead of accumulating across short-lived sync values.
	refs    uint32
	mu      psync.Mutex
	cond    psync.Cond
	waiters uint32
}

var semaOnce psync.Once
var semaMu psync.Mutex

// Pointer keys are cleared by map deletion; a uintptr key could leave stale
// address bits in an empty bucket and retain the Cond under conservative GC.
var semaMap map[*uint32]*semaState

func initSemaMap() {
	semaMu.Init(nil)
	semaMap = make(map[*uint32]*semaState)
}

func getSemaState(addr *uint32, create bool) *semaState {
	semaOnce.Do(initSemaMap)
	semaMu.Lock()
	st := semaMap[addr]
	if st == nil {
		if !create {
			semaMu.Unlock()
			return nil
		}
		st = &semaState{}
		st.mu.Init(nil)
		st.cond.Init(nil)
		semaMap[addr] = st
	}
	st.refs++
	semaMu.Unlock()
	return st
}

func releaseSemaState(addr *uint32, st *semaState) {
	semaMu.Lock()
	st.refs--
	idle := st.refs == 0
	if idle {
		delete(semaMap, addr)
	}
	semaMu.Unlock()
	if idle {
		st.cond.Destroy()
		st.mu.Destroy()
	}
}

func semaAcquire(addr *uint32) {
	if darwinUseAddressSema() {
		for {
			value := latomic.LoadUint32(addr)
			if value != 0 && latomic.CompareAndSwapUint32(addr, value, value-1) {
				return
			}
			if err := psync.WaitUint32(addr, 0); err != 0 && !psync.WaitErrnoRetryable(err) {
				// Avoid pinning a core in a tight spin if the kernel wait
				// fails persistently (e.g. ENOMEM/EFAULT).
				c.Usleep(100)
			}
		}
	}
	for {
		v := latomic.LoadUint32(addr)
		if v != 0 && latomic.CompareAndSwapUint32(addr, v, v-1) {
			return
		}
		st := getSemaState(addr, true)
		st.mu.Lock()
		for {
			v = latomic.LoadUint32(addr)
			if v != 0 {
				if latomic.CompareAndSwapUint32(addr, v, v-1) {
					st.mu.Unlock()
					releaseSemaState(addr, st)
					return
				}
				// Fast-path acquirers can change the count without st.mu. A
				// failed CAS does not mean it is zero: recheck before sleeping.
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
	st := getSemaState(addr, false)
	if st == nil {
		return
	}
	st.mu.Lock()
	if st.waiters != 0 {
		st.cond.Signal()
	}
	st.mu.Unlock()
	releaseSemaState(addr, st)
}
