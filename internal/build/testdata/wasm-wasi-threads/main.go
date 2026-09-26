package main

import (
	"sync"
	"time"
	_ "unsafe"
)

const LLGoFiles = "_wrap/stack.c"

//go:linkname workerStackBounds C.llgo_wasi_worker_stack_bounds
func workerStackBounds() int32

//go:linkname monotonicClock C.llgo_probe_monotonic_clock
func monotonicClock() int64

//go:linkname timerClock C.llgo_probe_timer_clock
func timerClock() int32

//go:linkname runtimeNanotime time.runtimeNano
func runtimeNanotime() int64

//go:linkname gmpForTesting github.com/xgo-dev/llgo/runtime/internal/runtime.GMPForTesting
func gmpForTesting() (goid, parentGoid uint64, mid int64, pid int32, gstatus, pstatus uint32, linked bool)

//go:linkname gStateForTesting github.com/xgo-dev/llgo/runtime/internal/runtime.GStateForTesting
func gStateForTesting() (count uint64, mainExited bool)

func main() {
	checkClocks()
	if workerStackBounds() != 1 {
		panic("WAMR main-thread C stack bounds unavailable")
	}
	_, _, mainMID, _, _, _, linked := gmpForTesting()
	if !linked {
		panic("main G/M/P is not linked")
	}
	// Start the timer service before measuring G retirement. It is a
	// persistent goroutine, so its count is part of the baseline.
	time.Sleep(time.Millisecond)
	baseline, _ := gStateForTesting()

	const workers = 4
	var wait sync.WaitGroup
	ids := make(chan int64, workers)
	release := make(chan struct{})
	for i := 0; i < workers; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			checkClocks()
			if workerStackBounds() != 1 {
				panic("WAMR pthread C stack bounds unavailable")
			}
			_, _, mid, _, _, _, linked := gmpForTesting()
			if !linked {
				panic("worker G/M/P is not linked")
			}
			ids <- mid
			<-release
			time.Sleep(time.Millisecond)
		}()
	}
	seen := map[int64]bool{mainMID: true}
	for i := 0; i < workers; i++ {
		mid := <-ids
		if seen[mid] {
			panic("goroutines reused a host M")
		}
		seen[mid] = true
	}
	close(release)
	wait.Wait()
	if len(seen) != workers+1 {
		panic("not all WASI pthreads ran")
	}

	// This checks Go bookkeeping only. WAMR releases host thread slots after
	// mexit; the runner reserves slots for every thread created by this probe.
	waitForBaseline(baseline)
	for round := 0; round < 12; round++ {
		var batch sync.WaitGroup
		values := make(chan *threadValue, workers)
		for i := 0; i < workers; i++ {
			batch.Add(1)
			go func(index int) {
				defer batch.Done()
				inner := &threadValue{value: round*workers + index}
				values <- &threadValue{value: inner.value + 1, next: inner}
			}(i)
		}
		batch.Wait()
		close(values)
		seenValues := make(map[int]bool, workers)
		for value := range values {
			if value == nil || value.next == nil || value.value != value.next.value+1 || seenValues[value.next.value] {
				panic("cross-thread pointer handoff failed")
			}
			seenValues[value.next.value] = true
		}
		for i := 0; i < workers; i++ {
			if !seenValues[round*workers+i] {
				panic("missing cross-thread pointer handoff")
			}
		}
		waitForBaseline(baseline)
	}
	println("wasi threads ok")
}

type threadValue struct {
	value int
	next  *threadValue
}

func waitForBaseline(baseline uint64) {
	deadline := time.Now().Add(5 * time.Second)
	for {
		count, exited := gStateForTesting()
		if count == baseline && !exited {
			return
		}
		if time.Now().After(deadline) {
			panic("WASI goroutine count did not return to baseline")
		}
		time.Sleep(time.Millisecond)
	}
}

func checkClocks() {
	before := monotonicClock()
	now := runtimeNanotime()
	after := monotonicClock()
	if before < 0 || now < before || now > after {
		panic("runtime nanotime does not use the WASI monotonic clock")
	}
	if timerClock() != 1 {
		panic("timer condition does not use the monotonic clock")
	}
	start := time.Now()
	time.Sleep(2 * time.Millisecond)
	if elapsed := monotonicClock() - after; elapsed < int64(2*time.Millisecond) || time.Since(start) < 2*time.Millisecond {
		panic("WASI monotonic timer returned early")
	}
}
