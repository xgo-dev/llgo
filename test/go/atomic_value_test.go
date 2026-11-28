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
		final := v.Load()
		got, ok := final.(int)
		if !ok || got < 1 || got > workers {
			t.Fatalf("round %d: final Value.Load() = %#v, want int in [1,%d]", r, final, workers)
		}
	}
}

func TestContextCancelConcurrentAtomicValue(t *testing.T) {
	const rounds = 150
	const workers = 8
	for r := 0; r < rounds; r++ {
		// Share a single cancelCtx across all workers so Done() and cancel()
		// concurrently race on the first store to cancelCtx.done (atomic.Value).
		ctx, cancel := context.WithCancel(context.Background())
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(workers)
		for w := 0; w < workers; w++ {
			doCancel := w%2 == 0
			go func() {
				defer wg.Done()
				<-start
				if doCancel {
					cancel()
				}
				<-ctx.Done()
				if ctx.Err() != context.Canceled {
					t.Errorf("ctx.Err() = %v, want context.Canceled", ctx.Err())
				}
			}()
		}
		close(start)
		wg.Wait()
	}
}
