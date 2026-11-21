// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// fiber scheduler; the golden output must stay identical on both. It only
// exercises sync paths that never park a goroutine; the contended paths
// (Mutex, WaitGroup.Wait with outstanding Dones, Cond) are covered by the
// syncmutex/syncrwcond/synconce fixtures, which park through the
// fiber-backed semaphore.
package main

import "sync"

func main() {
	var mu sync.Mutex
	mu.Lock()
	mu.Unlock()
	println("mutex ok")

	var once sync.Once
	calls := 0
	once.Do(func() { calls++ })
	once.Do(func() { calls++ })
	println("once", calls)

	var rw sync.RWMutex
	rw.Lock()
	rw.Unlock()
	rw.RLock()
	rw.RUnlock()
	println("rw ok")

	var wg sync.WaitGroup
	wg.Add(2)
	wg.Done()
	wg.Done()
	wg.Wait()
	println("wg ok")

	var m sync.Map
	m.Store("k", 3)
	v, ok := m.Load("k")
	println("map", v.(int), ok)
	_, ok = m.Load("none")
	println("miss", ok)
	n := 0
	m.Range(func(key, val any) bool { n++; return true })
	println("range", n)

	var pool sync.Pool
	pool.New = func() any { return 13 }
	println("pool", pool.Get().(int))
}
