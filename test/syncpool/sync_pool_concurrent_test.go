package syncpool

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

// Pool values may move between goroutines, including fibers sharing one OS
// thread. Check exclusive ownership and pool identity across a scheduling
// point without requiring Get to return the caller's most recent Put.
func TestSharedPoolsConcurrentOwnership(t *testing.T) {
	type item struct {
		pool  int
		owner int32
	}
	const workers = 32
	pools := make([]sync.Pool, 64)
	for i := range pools {
		i := i
		pools[i].New = func() any { return &item{pool: i} }
	}
	var ready, done sync.WaitGroup
	ready.Add(workers)
	done.Add(workers)
	release := make(chan struct{})
	for worker := 0; worker < workers; worker++ {
		go func(worker int) {
			defer done.Done()
			ready.Done()
			<-release
			for i := range pools {
				value := pools[i].Get().(*item)
				if value.pool != i {
					t.Errorf("worker %d, pool %d: value belongs to pool %d", worker, i, value.pool)
				}
				if !atomic.CompareAndSwapInt32(&value.owner, 0, int32(worker+1)) {
					t.Errorf("worker %d, pool %d: value already in use", worker, i)
					return
				}
				runtime.Gosched()
				if !atomic.CompareAndSwapInt32(&value.owner, int32(worker+1), 0) {
					t.Errorf("worker %d, pool %d: owner changed during use", worker, i)
					return
				}
				pools[i].Put(value)
			}
		}(worker)
	}
	ready.Wait()
	close(release)
	done.Wait()
}
