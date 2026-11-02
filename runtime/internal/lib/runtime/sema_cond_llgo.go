//go:build darwin || (linux && baremetal)

package runtime

import (
	"unsafe"

	latomic "sync/atomic"

	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
)

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
			if v != 0 {
				if latomic.CompareAndSwapUint32(addr, v, v-1) {
					st.mu.Unlock()
					return
				}
				// Fast-path acquirers can change the count without st.mu.
				// A failed CAS does not mean it is zero: recheck before sleeping.
				continue
			}
			st.waiters++
			st.cond.Wait(&st.mu)
			st.waiters--
		}
	}
}

func semaReleaseNative(addr *uint32) {
	latomic.AddUint32(addr, 1)
	st := getSemaState(addr)
	st.mu.Lock()
	if st.waiters != 0 {
		st.cond.Signal()
	}
	st.mu.Unlock()
}
