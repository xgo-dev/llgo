//go:build llgo && (!wasm || (wasip1 && llgo.wasi_threads))

package syncpool

import (
	"sync"
	"testing"
)

// LLGo's native and WASI pthread backends bind each goroutine to its own
// thread. JavaScript fibers can share a thread and its pool locals, so this
// retention check is limited to the pthread backends.
func TestSharedPoolsConcurrentThreadIsolation(t *testing.T) {
	const workers = 32
	pools := make([]sync.Pool, 64)
	var ready, done sync.WaitGroup
	ready.Add(workers)
	done.Add(workers)
	release := make(chan struct{})
	for worker := 0; worker < workers; worker++ {
		go func(worker int) {
			defer done.Done()
			for i := range pools {
				pools[i].Put(worker*len(pools) + i)
			}
			// Keep every worker alive with populated locals before any Get.
			ready.Done()
			<-release
			for i := range pools {
				want := worker*len(pools) + i
				if got := pools[i].Get(); got != want {
					t.Errorf("worker %d, pool %d: got %v, want %d", worker, i, got, want)
				}
			}
		}(worker)
	}
	ready.Wait()
	close(release)
	done.Wait()
}
