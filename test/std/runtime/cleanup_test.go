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
	var mu sync.Mutex
	cleaned := make(chan struct{}, 1)
	created := make(chan struct{})
	go func() {
		p := new([1024]byte)
		runtime.AddCleanup(p, func(struct{}) {
			mu.Lock()
			mu.Unlock()
			cleaned <- struct{}{}
		}, struct{}{})
		runtime.KeepAlive(p)
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
