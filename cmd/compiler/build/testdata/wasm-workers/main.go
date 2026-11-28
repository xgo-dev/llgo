package main

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	_ "unsafe"

	"github.com/xgo-dev/llgo/runtime/_test/workerlocality"
)

//go:linkname gmpForTesting github.com/xgo-dev/llgo/runtime/internal/runtime.GMPForTesting
func gmpForTesting() (goid, parentGoid uint64, mid int64, pid int32, gstatus, pstatus uint32, linked bool)

type workerIdentity struct {
	mid    int64
	pid    int32
	thread uintptr
}

func main() {
	testParallelWorkers()
	testPinnedGoroutine()
	testBoundedWorkerLifecycle()
	testCrossWorkerChannelHandoffs()
	testCrossWorkerSynchronization()
	testMixedSynchronizationWaiters()
	testInterleavedWorkerLocality()
	testCrossWorkerTimerWake()
	testRemoteWorkerGC()
	testConcurrentWorkerAllocation()
	println("wasm workers ok")
}

func testParallelWorkers() {
	var identities [2]workerIdentity
	record := func() {
		_, _, mid, pid, gstatus, pstatus, linked := gmpForTesting()
		if gstatus != 2 || pstatus != 1 || !linked {
			panic("invalid worker G/M/P state")
		}
		slot := parallelWorkerBarrier()
		if slot < 0 || int(slot) >= len(identities) {
			panic("parallel worker barrier timed out")
		}
		identities[slot] = workerIdentity{
			mid:    mid,
			pid:    pid,
			thread: parallelWorkerThread(slot),
		}
	}

	done := make(chan struct{})
	go func() {
		record()
		close(done)
	}()
	record()
	<-done

	left, right := identities[0], identities[1]
	if left.mid == right.mid || left.pid == right.pid {
		panic("goroutines did not run on distinct scheduler workers")
	}
	if left.thread == 0 || right.thread == 0 || left.thread == right.thread {
		panic("goroutines did not overlap on distinct pthreads")
	}
}

func testPinnedGoroutine() {
	done := make(chan struct{})
	go func() {
		_, _, mid, pid, _, _, linked := gmpForTesting()
		thread := currentWorkerThread()
		if !linked || thread == 0 {
			panic("invalid initial worker identity")
		}
		for range 32 {
			runtime.Gosched()
		}
		time.Sleep(time.Millisecond)
		_, _, currentMid, currentPid, _, _, currentLinked := gmpForTesting()
		if !currentLinked || currentMid != mid || currentPid != pid || currentWorkerThread() != thread {
			panic("started goroutine migrated between workers")
		}
		close(done)
	}()
	<-done
}

func testBoundedWorkerLifecycle() {
	const goroutines = 2_000
	workerCount := int(configuredWorkerCount())
	var (
		done   atomic.Uint32
		mid    int64
		thread uintptr
	)
	mids := make(map[int64]struct{})
	threads := make(map[uintptr]struct{})
	for i := uint32(1); i <= goroutines; i++ {
		go func() {
			_, _, currentMID, _, _, _, linked := gmpForTesting()
			currentThread := currentWorkerThread()
			if !linked || currentThread == 0 {
				panic("invalid lifecycle worker identity")
			}
			mid = currentMID
			thread = currentThread
			done.Store(i)
		}()
		for done.Load() != i {
			runtime.Gosched()
		}
		mids[mid] = struct{}{}
		threads[thread] = struct{}{}
	}
	if len(mids) != workerCount || len(threads) != workerCount {
		panic("goroutine lifecycle escaped the bounded worker pool")
	}
}

func testCrossWorkerChannelHandoffs() {
	// Keep this above the historical async callback stack-exhaustion threshold.
	const handoffs = 100_000
	values := make(chan int)
	done := make(chan struct{})
	go func() {
		for i := range handoffs {
			values <- i
		}
		close(done)
	}()
	for i := range handoffs {
		if value := <-values; value != i {
			panic("cross-worker channel handoff lost ordering")
		}
	}
	<-done
}

func testCrossWorkerSynchronization() {
	workerCount := int(configuredWorkerCount())
	goroutines := workerCount * 2
	var (
		counter atomic.Uint32
		mu      sync.Mutex
		wg      sync.WaitGroup
	)
	values := make(chan int, goroutines)
	workers := make(map[int64]int)
	wg.Add(goroutines)
	for i := range goroutines {
		go func() {
			_, _, mid, _, _, _, _ := gmpForTesting()
			mu.Lock()
			counter.Add(1)
			workers[mid]++
			mu.Unlock()
			values <- i
			wg.Done()
		}()
	}
	wg.Wait()
	close(values)

	seen := 0
	for range values {
		seen++
	}
	if seen != goroutines || counter.Load() != uint32(goroutines) {
		panic("cross-worker synchronization lost work")
	}
	if len(workers) != workerCount {
		panic("goroutines did not use the bounded worker pool")
	}
}

func testMixedSynchronizationWaiters() {
	// Mutexes/semaphores and Cond/notifyList use different queue maps but share
	// the runtime waiter cache. Wake both kinds together so reuse is exercised
	// while separate physical workers are returning waiter records.
	const (
		rounds  = 16
		waiters = 8
	)
	for range rounds {
		var contended sync.Mutex
		contended.Lock()
		started := make(chan struct{}, waiters)
		done := make(chan struct{}, waiters*2)
		for range waiters {
			go func() {
				started <- struct{}{}
				contended.Lock()
				contended.Unlock()
				done <- struct{}{}
			}()
		}

		cond := sync.NewCond(new(sync.Mutex))
		ready := make(chan struct{})
		waiting := 0
		released := false
		for range waiters {
			go func() {
				cond.L.Lock()
				waiting++
				if waiting == waiters {
					close(ready)
				}
				for !released {
					cond.Wait()
				}
				cond.L.Unlock()
				done <- struct{}{}
			}()
		}

		for range waiters {
			<-started
		}
		<-ready
		// Taking cond.L after the ready close guarantees the last waiter has
		// completed Cond.Wait's atomic enqueue-and-unlock transition.
		cond.L.Lock()
		released = true
		contended.Unlock()
		cond.Broadcast()
		cond.L.Unlock()
		for range waiters * 2 {
			<-done
		}
	}
}

type localityWaiter struct {
	mid     int64
	value   int
	release chan struct{}
	done    chan struct{}
}

func testInterleavedWorkerLocality() {
	workerCount := int(configuredWorkerCount())
	ready := make(chan localityWaiter, workerCount*2)
	nextValue := 0
	startBatch := func() {
		for range workerCount {
			nextValue++
			value := nextValue
			release := make(chan struct{})
			done := make(chan struct{})
			go func() {
				if workerlocality.CurrentLocalState() != nil {
					panic("goroutine-local state was inherited")
				}
				if state := workerlocality.CurrentInitializedState(); state == nil || *state != 0 {
					panic("initialized goroutine-local state was inherited")
				}
				workerlocality.SetLocalState(&value)
				workerlocality.SetInitializedState(value)
				_, _, mid, _, _, _, _ := gmpForTesting()
				ready <- localityWaiter{mid: mid, value: value, release: release, done: done}
				<-release
				if state := workerlocality.CurrentLocalState(); state == nil || *state != value {
					panic("goroutine-local state changed after an interleaved G ran")
				}
				if state := workerlocality.CurrentInitializedState(); state == nil || *state != value {
					panic("initialized goroutine-local state changed after an interleaved G ran")
				}
				close(done)
			}()
		}
	}

	byWorker := make(map[int64][]localityWaiter, workerCount)
	for range 2 {
		startBatch()
		for range workerCount {
			waiter := <-ready
			byWorker[waiter.mid] = append(byWorker[waiter.mid], waiter)
		}
	}
	if len(byWorker) != workerCount {
		panic("locality test did not cover every worker")
	}
	for _, waiters := range byWorker {
		if len(waiters) != 2 {
			panic("locality test did not interleave two goroutines per worker")
		}
		if waiters[0].value == waiters[1].value {
			panic("locality test reused a goroutine value")
		}
		close(waiters[0].release)
		<-waiters[0].done
		close(waiters[1].release)
		<-waiters[1].done
	}
}

func testCrossWorkerTimerWake() {
	done := make(chan struct{})
	go func() {
		time.Sleep(5 * time.Millisecond)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		panic("timer did not wake a worker")
	}
}

type workerGCPayload struct {
	value uint64
	pad   [32]byte
}

func testRemoteWorkerGC() {
	const want = uint64(0x76543210)
	live := &workerGCPayload{value: want}
	_, _, mainMID, _, _, _, _ := gmpForTesting()

	for attempts := 0; attempts < int(configuredWorkerCount())*2; attempts++ {
		start := make(chan bool)
		ready := make(chan int64)
		var (
			done   atomic.Bool
			result atomic.Uint64
		)
		go func() {
			_, _, mid, _, _, _, _ := gmpForTesting()
			ready <- mid
			if !<-start {
				done.Store(true)
				return
			}
			remoteLive := &workerGCPayload{value: want}
			runtime.GC()
			result.Store(remoteLive.value)
			done.Store(true)
		}()

		if mid := <-ready; mid == mainMID {
			start <- false
			for !done.Load() {
			}
			continue
		}
		start <- true
		for !done.Load() {
			if live.value != want {
				panic("remote GC lost an active worker root")
			}
		}
		if result.Load() != want || live.value != want {
			panic("remote worker GC lost a live root")
		}
		return
	}
	panic("remote GC did not run on another worker")
}

func testConcurrentWorkerAllocation() {
	const (
		goroutines = 4
		iterations = 512
		liveCount  = 16
	)
	start := make(chan struct{})
	done := make(chan int64, goroutines)
	for id := range goroutines {
		go func() {
			<-start
			var live [liveCount]*workerGCPayload
			for i := range iterations {
				slot := i % len(live)
				live[slot] = &workerGCPayload{value: uint64(id*iterations + i + 1)}
				if i == iterations/2 {
					runtime.GC()
				}
			}
			for _, value := range live {
				if value == nil || value.value == 0 {
					panic("concurrent allocation lost a live object")
				}
			}
			_, _, mid, _, _, _, _ := gmpForTesting()
			done <- mid
		}()
	}
	close(start)

	workers := make(map[int64]bool)
	for range goroutines {
		workers[<-done] = true
	}
	if len(workers) < 2 {
		panic("concurrent GC did not cover multiple workers")
	}
}
