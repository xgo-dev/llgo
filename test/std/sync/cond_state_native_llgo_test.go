//go:build llgo && go1.27 && !nogc && (!wasm || llgo.wasi_threads)

package sync_test

import (
	"sync"
	"testing"
	"testing/synctest"
	_ "unsafe"
)

//go:linkname notifyStateCount runtime.syncNotifyStateCountForTesting
func notifyStateCount() int

func TestCondStateReclamation(t *testing.T) {
	const count = 32
	before := notifyStateCount()
	synctest.Test(t, func(t *testing.T) {
		conds := make([]*sync.Cond, count)
		for i := range conds {
			cond := sync.NewCond(new(sync.Mutex))
			conds[i] = cond
			go func() {
				cond.L.Lock()
				cond.Wait()
				cond.L.Unlock()
			}()
		}
		synctest.Wait()
		if got := notifyStateCount(); got != before+count {
			t.Fatalf("parked notification states = %d, want %d", got, before+count)
		}
		for _, cond := range conds {
			cond.Broadcast()
		}
		synctest.Wait()
	})
	if got := notifyStateCount(); got != before {
		t.Fatalf("discarded notification states remain: %d, want %d", got, before)
	}
}
