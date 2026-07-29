package main

import (
	"sync"
	"sync/atomic"
)

func main() {
	values := make(chan int)
	var wg sync.WaitGroup
	var mu sync.Mutex
	wg.Add(1)
	go func() {
		mu.Lock()
		values <- 42
		mu.Unlock()
		wg.Done()
	}()
	if value := <-values; value != 42 {
		panic("channel value mismatch")
	}
	wg.Wait()

	var rw sync.RWMutex
	rw.RLock()
	rw.RUnlock()
	rw.Lock()
	rw.Unlock()

	cond := sync.NewCond(&mu)
	started := make(chan struct{})
	ready := false
	wg.Add(1)
	go func() {
		mu.Lock()
		close(started)
		for !ready {
			cond.Wait()
		}
		mu.Unlock()
		wg.Done()
	}()
	<-started
	mu.Lock()
	ready = true
	cond.Signal()
	mu.Unlock()
	wg.Wait()

	var once sync.Once
	once.Do(func() {})
	var pool sync.Pool
	pool.Put("value")
	if pool.Get() != "value" {
		panic("sync.Pool value mismatch")
	}
	var atomicValue atomic.Value
	atomicValue.Store("value")
	if atomicValue.Load() != "value" {
		panic("atomic.Value mismatch")
	}
	println("wasm blocking primitives ok")
}
