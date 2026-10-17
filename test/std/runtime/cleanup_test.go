//go:build go1.24 && !baremetal && !nogc && !wasm

package runtime_test

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

// Collection must not execute a user cleanup on the collecting goroutine:
// that goroutine may already hold a lock needed by the cleanup.
func TestCleanupDoesNotBlockCollector(t *testing.T) {
	// BDWGC may conservatively retain one object through a stale stack or
	// register value. Require a cleanup from several independent candidates.
	const candidates = 8
	var mu sync.Mutex
	cleaned := make(chan struct{}, candidates)
	created := make(chan struct{})
	go func() {
		// Allocation-triggered collections must obey the same lock constraint
		// as the explicit collections below.
		mu.Lock()
		for range candidates {
			p := new([1024]byte)
			runtime.AddCleanup(p, func(struct{}) {
				mu.Lock()
				mu.Unlock()
				cleaned <- struct{}{}
			}, struct{}{})
			runtime.KeepAlive(p)
		}
		mu.Unlock()
		close(created)
	}()
	<-created
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		runtime.GC()
		mu.Unlock()
		select {
		case <-cleaned:
			return
		default:
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("cleanup did not run")
}
