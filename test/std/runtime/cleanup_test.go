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
	const probes = 16
	var mu sync.Mutex
	cleaned := make(chan struct{}, probes)
	created := make(chan struct{}, probes)
	// A conservative collector can retain an individual allocation because
	// of a stale stack/register value. Use independent creator stacks so one
	// such value cannot prevent this worker-dispatch test from observing any
	// cleanup. Hold mu during creation too: an automatic collection must not
	// execute a callback inline on a creator before our explicit GC calls.
	mu.Lock()
	for range probes {
		go func() {
			p := new([1024]byte)
			runtime.AddCleanup(p, func(struct{}) {
				mu.Lock()
				mu.Unlock()
				cleaned <- struct{}{}
			}, struct{}{})
			runtime.KeepAlive(p)
			created <- struct{}{}
		}()
	}
	for range probes {
		<-created
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		runtime.GC()
		mu.Unlock()
		select {
		case <-cleaned:
			return
		default:
		}
		time.Sleep(time.Millisecond)
		mu.Lock()
	}
	mu.Unlock()
	t.Fatal("cleanup did not run")
}
