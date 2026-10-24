//go:build go1.27

package synctest_test

import (
	"runtime"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

func TestFakeClockAndWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		if start.Unix() != 946684800 {
			t.Fatalf("start = %v", start)
		}
		done := make(chan int)
		go func() { time.Sleep(10 * time.Hour); done <- 42 }()
		synctest.Wait()
		if time.Since(start) != 0 {
			t.Fatal("Wait advanced time")
		}
		select {
		case <-done:
			t.Fatal("future sleep completed")
		default:
		}
		if <-done != 42 {
			t.Fatal("lost send")
		}
		if time.Since(start) != 10*time.Hour {
			t.Fatalf("elapsed = %v", time.Since(start))
		}
	})
}

func TestFakeClockOutsideTimeArithmetic(t *testing.T) {
	outside := time.Now()
	synctest.Test(t, func(t *testing.T) {
		inside := time.Now()
		// Round(0) removes monotonic readings without changing wall time.
		want := inside.Round(0).Sub(outside.Round(0))
		if got := inside.Sub(outside); got != want {
			t.Fatalf("inside.Sub(outside) = %v, wall-clock difference = %v", got, want)
		}
		if got := outside.Sub(inside); got != -want {
			t.Fatalf("outside.Sub(inside) = %v, want %v", got, -want)
		}
		if got := time.Since(outside); got != want {
			t.Fatalf("Since(outside) = %v, want %v", got, want)
		}
	})
}

func TestTimerStopResetAndTicker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		timer := time.NewTimer(time.Hour)
		if !timer.Stop() {
			t.Fatal("Stop returned false")
		}
		if timer.Reset(2 * time.Hour) {
			t.Fatal("Reset of stopped timer returned true")
		}
		tick := time.NewTicker(30 * time.Minute)
		defer tick.Stop()
		for i := 1; i <= 3; i++ {
			if got := <-tick.C; got.Sub(start) != time.Duration(i)*30*time.Minute {
				t.Fatalf("tick = %v", got)
			}
		}
		tick.Stop()
		if got := <-timer.C; got.Sub(start) != 2*time.Hour {
			t.Fatalf("timer = %v", got)
		}
		fired := make(chan struct{})
		time.AfterFunc(time.Hour, func() {
			if time.Now().UTC().Year() != 2000 {
				panic("timer callback lost bubble")
			}
			close(fired)
		})
		<-fired
		if time.Since(start) != 3*time.Hour {
			t.Fatal(time.Since(start))
		}
	})
}

func TestWaitGroupAndCondDurable(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var mu sync.Mutex
		cond := sync.NewCond(&mu)
		ready := make(chan struct{}, 3)
		var wg sync.WaitGroup
		for i := 0; i < 3; i++ {
			wg.Go(func() { mu.Lock(); ready <- struct{}{}; cond.Wait(); mu.Unlock() })
		}
		for i := 0; i < 3; i++ {
			<-ready
		}
		synctest.Wait()
		if len(ready) != 0 {
			t.Fatal("unexpected ready")
		}
		mu.Lock()
		cond.Signal()
		mu.Unlock()
		synctest.Wait()
		mu.Lock()
		cond.Broadcast()
		mu.Unlock()
		wg.Wait()
	})
}

func TestOutsideWaitsKeepClockActive(t *testing.T) {
	var mu sync.Mutex
	mu.Lock()
	outside := make(chan struct{})
	go func() {
		time.Sleep(20 * time.Millisecond)
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		close(outside)
	}()
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		future := make(chan struct{})
		go func() { time.Sleep(time.Hour); close(future) }()
		mu.Lock()
		mu.Unlock()
		if time.Since(start) != 0 {
			t.Fatal("mutex wait was durable")
		}
		<-outside
		if time.Since(start) != 0 {
			t.Fatal("outside channel wait was durable")
		}
		<-future
		if time.Since(start) != time.Hour {
			t.Fatal(time.Since(start))
		}
	})
}

func TestConcurrentIndependentBubbles(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Go(func() {
			synctest.Test(t, func(t *testing.T) {
				start := time.Now()
				time.Sleep(time.Hour)
				if time.Since(start) != time.Hour {
					t.Fatal("shared fake clock")
				}
			})
		})
	}
	wg.Wait()
}

func TestRecoveredPanicAndCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer func() {
				if recover() == nil {
					panic("missing panic")
				}
			}()
			panic("recovered child")
		}()
		<-done
		object := new([64]byte)
		cleanup := runtime.AddCleanup(object, func(int) {}, 1)
		cleanup.Stop()
		runtime.SetFinalizer(object, func(*[64]byte) {})
		runtime.SetFinalizer(object, nil)
		runtime.KeepAlive(object)
		synctest.Wait()
	})
	synctest.Test(t, func(t *testing.T) { time.Sleep(time.Hour) })
}

var globalWaitGroup sync.WaitGroup

func TestGlobalWaitGroupRemainsUnbubbled(t *testing.T) {
	request := make(chan struct{})
	outsideDone := make(chan struct{})
	go func() { defer close(outsideDone); <-request; time.Sleep(20 * time.Millisecond); globalWaitGroup.Done() }()
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		future := make(chan struct{})
		go func() { time.Sleep(time.Hour); close(future) }()
		globalWaitGroup.Add(1)
		request <- struct{}{}
		globalWaitGroup.Wait()
		if time.Since(start) != 0 {
			t.Fatal("global WaitGroup became durable")
		}
		<-future
	})
	<-outsideDone
}

// WebAssembly timer polling can run on a scheduler context with no current G.
// Its sendTime callback must retain ordinary clock behavior outside a bubble.
func TestTimerOutsideBubble(t *testing.T) {
	before := time.Now()
	timer := time.NewTimer(time.Millisecond)
	defer timer.Stop()
	got := <-timer.C
	if got.Before(before) || got.After(time.Now()) {
		t.Fatalf("outside timer timestamp=%v", got)
	}
}

func TestWaitGroupAfterHeapGrowth(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// WASI may grow into disjoint GC arenas after its initial 1 MiB arena.
		// Put the WaitGroup beyond that range in one escaped allocation.
		object := new(struct {
			padding [2 << 20]byte
			wg      sync.WaitGroup
		})
		start := time.Now()
		object.wg.Add(1)
		go func() { time.Sleep(time.Hour); object.wg.Done() }()
		object.wg.Wait()
		if elapsed := time.Since(start); elapsed != time.Hour {
			t.Fatalf("large-allocation WaitGroup elapsed = %v", elapsed)
		}
		runtime.KeepAlive(object)
	})
}
