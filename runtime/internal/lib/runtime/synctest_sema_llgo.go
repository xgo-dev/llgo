//go:build darwin || linux || windows || (llgo && wasip1 && wasm && llgo.wasi_threads)

package runtime

import (
	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
	"sync/atomic"
)

var synctestSemaOnce psync.Once
var synctestSemaMu psync.Mutex
var synctestSemaWaiters map[*uint32][]*rt.SynctestWaiter

func initSynctestSema() {
	synctestSemaMu.Init(nil)
	synctestSemaWaiters = make(map[*uint32][]*rt.SynctestWaiter)
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
	synctestSemaWaiters[addr] = append(synctestSemaWaiters[addr], w)
	w.Block()
	synctestSemaMu.Unlock()
	semaAcquire(addr)
	synctestSemaMu.Lock()
	w.Wake()
	waiters := synctestSemaWaiters[addr]
	for i, entry := range waiters {
		if entry == w {
			waiters = append(waiters[:i], waiters[i+1:]...)
			break
		}
	}
	if len(waiters) == 0 {
		delete(synctestSemaWaiters, addr)
	} else {
		synctestSemaWaiters[addr] = waiters
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
	for _, w := range synctestSemaWaiters[addr] {
		w.CheckWake()
	}
	for _, w := range synctestSemaWaiters[addr] {
		w.Wake()
	}
	// Keep publication under the registration lock: a waiter must either
	// consume an existing token, or be made active before a new token appears.
	semaReleaseNative(addr)
}
