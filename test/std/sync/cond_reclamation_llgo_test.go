//go:build llgo && go1.27 && !nogc

package sync_test

import (
	"runtime"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

func TestCondStateReclamation(t *testing.T) {
	const count = 32
	collected := make(chan struct{}, count)
	exerciseCondStates(t, collected, count)
	deadline := time.Now().Add(5 * time.Second)
	for len(collected) != count && time.Now().Before(deadline) {
		runtime.GC()
		time.Sleep(time.Millisecond)
	}
	if got := len(collected); got != count {
		t.Fatalf("reclaimed %d/%d discarded condition variables", got, count)
	}
}

type collectibleCond struct {
	sync.Cond
	marker *[64]byte
}

//go:noinline
func exerciseCondStates(t *testing.T, collected chan struct{}, count int) {
	synctest.Test(t, func(t *testing.T) {
		conds := make([]*collectibleCond, count)
		for i := range conds {
			cond := &collectibleCond{Cond: sync.Cond{L: new(sync.Mutex)}, marker: new([64]byte)}
			conds[i] = cond
			// Cond contains a self-pointer in its copy checker. Watch its pointer-free
			// marker to avoid an ordered-finalization cycle in conservative collectors.
			runtime.AddCleanup(cond.marker, func(done chan struct{}) { done <- struct{}{} }, collected)
			go func() {
				cond.L.Lock()
				cond.Wait()
				cond.L.Unlock()
				runtime.KeepAlive(cond.marker)
			}()
		}
		// Real parked waiters ensure an idle-notify fast path cannot bypass the
		// states whose keys would otherwise keep these condition variables alive.
		synctest.Wait()
		for _, cond := range conds {
			cond.Broadcast()
		}
		synctest.Wait()
	})
}
