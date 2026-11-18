//go:build darwin || linux || windows || (llgo && wasip1 && wasm && llgo.wasi_threads)

package runtime

import (
	"sync/atomic"
	"unsafe"

	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
	"github.com/xgo-dev/llgo/runtime/internal/wordmap"
)

var synctestSemaOnce psync.Once
var synctestSemaMu psync.Mutex
var synctestSemaWaiters wordmap.Map[unsafe.Pointer, []*rt.SynctestWaiter]

func initSynctestSema() {
	synctestSemaMu.Init(nil)
}

func semaAcquireWaitGroup(addr *uint32, durable bool) {
	if !durable || rt.CurrentSynctestBubble() == nil {
		semaAcquire(addr)
		return
	}
	synctestSemaOnce.Do(initSynctestSema)
	synctestSemaMu.Lock()
	for {
		n := atomic.LoadUint32(addr)
		if n == 0 {
			break
		}
		if atomic.CompareAndSwapUint32(addr, n, n-1) {
			synctestSemaMu.Unlock()
			return
		}
	}
	w := rt.NewSynctestWaiter()
	synctestSemaWaiters.Set(unsafe.Pointer(addr), append(synctestSemaWaiters.Get(unsafe.Pointer(addr)), w))
	w.Block()
	synctestSemaMu.Unlock()
	semaAcquire(addr)
	synctestSemaMu.Lock()
	w.Wake()
	waiters := synctestSemaWaiters.Get(unsafe.Pointer(addr))
	for i, entry := range waiters {
		if entry == w {
			waiters = append(waiters[:i], waiters[i+1:]...)
			break
		}
	}
	if len(waiters) == 0 {
		synctestSemaWaiters.Delete(unsafe.Pointer(addr))
	} else {
		synctestSemaWaiters.Set(unsafe.Pointer(addr), waiters)
	}
	synctestSemaMu.Unlock()
}

func semaRelease(addr *uint32) {
	if rt.CurrentSynctestBubble() == nil && !rt.SynctestActive() {
		semaReleaseNative(addr)
		return
	}
	synctestSemaOnce.Do(initSynctestSema)
	synctestSemaMu.Lock()
	defer synctestSemaMu.Unlock()
	for _, w := range synctestSemaWaiters.Get(unsafe.Pointer(addr)) {
		w.CheckWake()
	}
	for _, w := range synctestSemaWaiters.Get(unsafe.Pointer(addr)) {
		w.Wake()
	}
	// Keep publication under the registration lock: a waiter must either
	// consume an existing token, or be made active before a new token appears.
	semaReleaseNative(addr)
}
