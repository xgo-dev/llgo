package sync_test

import (
	"sync"
	"testing"
	"time"
)

// Basic tests that should work with both Go and LLGo
func TestMutexBasic(t *testing.T) {
	var mu sync.Mutex

	mu.Lock()
	mu.Unlock()
	// Should be able to lock again
	mu.Lock()
	mu.Unlock()
}

func TestRWMutexBasic(t *testing.T) {
	var rwMu sync.RWMutex

	// Test read/write lock operations
	rwMu.Lock()
	rwMu.Unlock()

	rwMu.RLock()
	rwMu.RUnlock()
}

func TestOnceBasic(t *testing.T) {
	var once sync.Once
	var called bool

	once.Do(func() {
		called = true
	})

	if !called {
		t.Fatal("Once.Do function was not called")
	}

	// Second call should not execute the function
	once.Do(func() {
		t.Fatal("Once.Do function was called twice")
	})
}

func TestWaitGroupBasic(t *testing.T) {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		time.Sleep(10 * time.Millisecond)
		wg.Done()
	}()

	wg.Wait()

	// Add to completed WaitGroup and wait again
	wg.Add(1)
	go func() {
		time.Sleep(10 * time.Millisecond)
		wg.Done()
	}()

	wg.Wait()
}

func TestPoolSkip(t *testing.T) {
	t.Skip("Pool has type assertion issues in LLGo runtime")
}

func TestCondSkip(t *testing.T) {
	t.Skip("Cond has type assertion issues in LLGo runtime")
}

func TestConcurrentAccess(t *testing.T) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	counter := 0

	// Test concurrent access to mutex
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			mu.Lock()
			counter++
			mu.Unlock()
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	if counter != 10 {
		t.Fatalf("Expected counter = 10, got %d", counter)
	}
}

func TestLockerInterface(t *testing.T) {
	var mu sync.Mutex
	var rwMu sync.RWMutex

	// Test that both Mutex and RWMutex implement Locker interface
	var l sync.Locker = &mu
	l.Lock()
	l.Unlock()

	l = &rwMu
	l.Lock()
	l.Unlock()
}
