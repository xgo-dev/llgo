package sync_test

import (
	"sync"
	"testing"
	"time"
)

func BenchmarkCondNotifyIdle(b *testing.B) {
	cond := sync.NewCond(new(sync.Mutex))
	for _, op := range []struct {
		name   string
		notify func()
	}{{"Signal", cond.Signal}, {"Broadcast", cond.Broadcast}} {
		b.Run(op.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				op.notify()
			}
		})
	}
}

func TestCondBasic(t *testing.T) {
	var mu sync.Mutex
	cond := sync.NewCond(&mu)

	// Test basic Cond operations
	cond.Signal()
	cond.Broadcast()

	// Test with waiting goroutine
	signaled := false
	go func() {
		mu.Lock()
		for !signaled {
			cond.Wait()
		}
		mu.Unlock()
	}()

	time.Sleep(10 * time.Millisecond)
	mu.Lock()
	signaled = true
	cond.Signal()
	mu.Unlock()

	// Wait for goroutine to complete
	time.Sleep(50 * time.Millisecond)
	if !signaled {
		t.Fatal("Cond.Signal failed to wake up waiting goroutine")
	}
}

func TestCondBroadcast(t *testing.T) {
	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	ready := make(chan struct{}, 3)
	var done sync.WaitGroup
	done.Add(3)

	// Start multiple waiting goroutines
	wokenUp := 0
	for i := 0; i < 3; i++ {
		go func() {
			defer done.Done()
			mu.Lock()
			ready <- struct{}{}
			cond.Wait()
			wokenUp++
			mu.Unlock()
		}()
	}

	// Wait until every goroutine has acquired the lock immediately before
	// Wait. Taking the lock below then guarantees they have all entered Wait.
	for i := 0; i < 3; i++ {
		<-ready
	}

	// Broadcast to wake all
	mu.Lock()
	cond.Broadcast()
	mu.Unlock()
	done.Wait()

	mu.Lock()
	finalWoken := wokenUp
	mu.Unlock()

	if finalWoken != 3 {
		t.Fatalf("Expected all 3 goroutines to be awakened, but only %d woke up", finalWoken)
	}
}

func TestCondWait(t *testing.T) {
	var mu sync.Mutex
	cond := sync.NewCond(&mu)

	// Test basic Wait functionality
	woken := false
	mu.Lock()
	go func() {
		mu.Lock()
		woken = true
		cond.Signal()
		mu.Unlock()
	}()

	for !woken {
		cond.Wait()
	}
	mu.Unlock()

	if !woken {
		t.Fatal("Wait should have returned after Signal")
	}
}
