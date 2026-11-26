package gotest

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

func TestAtomicValueConcurrentFirstStore(t *testing.T) {
	const rounds = 200
	const workers = 8
	for r := 0; r < rounds; r++ {
		var v atomic.Value
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(workers)
		for w := 0; w < workers; w++ {
			val := w + 1
			go func() {
				defer wg.Done()
				<-start
				switch val % 3 {
				case 0:
					v.Store(val)
				case 1:
					_ = v.Swap(val)
				default:
					_ = v.CompareAndSwap(nil, val)
				}
				got := v.Load()
				if got != nil {
					if n, ok := got.(int); !ok || n < 1 || n > workers {
						t.Errorf("unexpected loaded value %#v", got)
					}
				}
			}()
		}
		close(start)
		wg.Wait()
		got, ok := v.Load().(int)
		if !ok || got < 1 || got > workers {
			t.Fatalf("round %d: final Value.Load() = %#v, want int in [1,%d]", r, v.Load(), workers)
		}
	}
}

func TestContextCancelConcurrentAtomicValue(t *testing.T) {
	const workers = 8
	const iters = 250
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				if ctx.Err() != context.Canceled {
					t.Errorf("ctx.Err() = %v, want context.Canceled", ctx.Err())
					return
				}
				select {
				case <-ctx.Done():
				default:
					t.Errorf("ctx.Done() not closed after cancel")
					return
				}
			}
		}()
	}
	wg.Wait()
}
