package main

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

//go:linkname currentGForTesting github.com/goplus/llgo/runtime/internal/runtime.CurrentGForTesting
func currentGForTesting() unsafe.Pointer

//go:linkname parkForTesting github.com/goplus/llgo/runtime/internal/runtime.ParkForTesting
func parkForTesting()

//go:linkname readyForTesting github.com/goplus/llgo/runtime/internal/runtime.ReadyForTesting
func readyForTesting(unsafe.Pointer)

//go:linkname schedulerStateForTesting github.com/goplus/llgo/runtime/internal/runtime.SchedulerStateForTesting
func schedulerStateForTesting() (runq uintptr, mid int64, pid int32)

//go:linkname gmpForTesting github.com/goplus/llgo/runtime/internal/runtime.GMPForTesting
func gmpForTesting() (goid, parentGoid uint64, mid int64, pid int32, gstatus, pstatus uint32, linked bool)

var (
	parked     unsafe.Pointer
	mainMID    int64
	mainPID    int32
	mainGID    uint64
	seenG      [4]uint64
	seenGCount int
	eventLog   [8]int
	eventCount int
	done       int
	lifecycle  int
)

func event(value int) {
	eventLog[eventCount] = value
	eventCount++
}

func checkCurrentG() {
	goid, parent, mid, pid, gstatus, pstatus, linked := gmpForTesting()
	if goid == 0 || goid == mainGID || parent != mainGID {
		panic("invalid goroutine identity")
	}
	if mid != mainMID || pid != mainPID {
		panic("goroutine did not reuse the single worker M/P")
	}
	if gstatus != 2 || pstatus != 1 || !linked {
		panic("invalid running G/M/P state")
	}
	for i := 0; i < seenGCount; i++ {
		if seenG[i] == goid {
			panic("duplicate goroutine identity")
		}
	}
	seenG[seenGCount] = goid
	seenGCount++
}

func main() {
	checkWasmModel()
	if schedulerDeadlockMode() != 0 {
		testParkedMainDeadlock()
		return
	}
	var (
		gstatus uint32
		pstatus uint32
		linked  bool
	)
	mainGID, _, mainMID, mainPID, gstatus, pstatus, linked = gmpForTesting()
	if mainGID != 1 || mainMID != 1 || mainPID != 0 || gstatus != 2 || pstatus != 1 || !linked {
		panic("invalid main G/M/P state")
	}

	go func() {
		checkCurrentG()
		event(1)
		parked = currentGForTesting()
		parkForTesting()
		event(8)
		done++
	}()

	go func() {
		checkCurrentG()
		event(2)
		runtime.Gosched()
		event(6)
		readyForTesting(parked)
		event(7)
		done++
	}()

	go func() {
		checkCurrentG()
		defer func() {
			if recover() != "expected panic" {
				panic("unexpected recover value")
			}
			event(3)
			done++
		}()
		panic("expected panic")
	}()

	go func() {
		checkCurrentG()
		defer func() {
			event(4)
			done++
		}()
		runtime.Goexit()
		panic("Goexit returned")
	}()

	if runq, mid, pid := schedulerStateForTesting(); runq != 4 || mid != mainMID || pid != mainPID {
		panic("invalid initial scheduler state")
	}
	event(0)
	for done != 4 {
		runtime.Gosched()
	}

	want := [...]int{0, 1, 2, 3, 4, 6, 7, 8}
	if eventCount != len(want) {
		panic("unexpected event count")
	}
	for i, value := range want {
		if eventLog[i] != value {
			panic("unexpected scheduler order")
		}
	}
	if seenGCount != len(seenG) {
		panic("not all goroutines ran")
	}
	testGoroutineLifecycle()
	testBlockingPrimitives()
	println("wasm scheduler ok")
}

func testBlockingPrimitives() {
	testChannelsAndSelect()
	testWaitGroup()
	testMutexes()
	testCond()
	testSyncHelpers()
	testSyncMap()
}

func testChannelsAndSelect() {
	values := make(chan int)
	ack := make(chan struct{})
	go func() {
		values <- 41
		close(ack)
	}()
	if value := <-values; value != 41 {
		panic("unexpected channel value")
	}
	<-ack

	buffered := make(chan int, 2)
	buffered <- 1
	buffered <- 2
	if len(buffered) != 2 || cap(buffered) != 2 {
		panic("buffered channel size mismatch")
	}
	if <-buffered != 1 || <-buffered != 2 {
		panic("buffered channel order mismatch")
	}

	left := make(chan int)
	right := make(chan int)
	go func() {
		right <- 42
	}()
	select {
	case <-left:
		panic("select chose a blocked channel")
	case value := <-right:
		if value != 42 {
			panic("unexpected select value")
		}
	}

	selected := make(chan int)
	go func() {
		select {
		case value := <-left:
			selected <- value
		case value := <-right:
			selected <- value
		}
	}()
	runtime.Gosched()
	left <- 43
	if value := <-selected; value != 43 {
		panic("blocked select chose the wrong channel")
	}

	select {
	case <-right:
		panic("non-blocking select chose a blocked channel")
	default:
	}

	close(right)
	if value, ok := <-right; value != 0 || ok {
		panic("closed channel receive mismatch")
	}
}

func testWaitGroup() {
	var wg sync.WaitGroup
	count := 0
	wg.Add(2)
	go func() {
		count++
		wg.Done()
	}()
	go func() {
		runtime.Gosched()
		count++
		wg.Done()
	}()
	wg.Wait()
	if count != 2 {
		panic("WaitGroup returned too early")
	}

	wg.Add(1)
	go wg.Done()
	wg.Wait()
}

func testMutexes() {
	var mu sync.Mutex
	started := make(chan struct{})
	finished := make(chan struct{})
	value := 0
	mu.Lock()
	go func() {
		close(started)
		mu.Lock()
		value = 1
		mu.Unlock()
		close(finished)
	}()
	<-started
	mu.Unlock()
	<-finished
	if value != 1 {
		panic("Mutex waiter did not run")
	}
	if !mu.TryLock() {
		panic("Mutex.TryLock failed")
	}
	mu.Unlock()

	var rw sync.RWMutex
	rw.RLock()
	finished = make(chan struct{})
	go func() {
		rw.Lock()
		value = 2
		rw.Unlock()
		close(finished)
	}()
	runtime.Gosched()
	rw.RUnlock()
	<-finished
	if value != 2 {
		panic("RWMutex waiter did not run")
	}
}

func testCond() {
	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	arrived := make(chan struct{}, 2)
	done := make(chan struct{}, 2)
	ready := false
	for i := 0; i < 2; i++ {
		go func() {
			mu.Lock()
			arrived <- struct{}{}
			for !ready {
				cond.Wait()
			}
			mu.Unlock()
			done <- struct{}{}
		}()
	}
	<-arrived
	<-arrived

	mu.Lock()
	ready = true
	cond.Signal()
	mu.Unlock()
	<-done
	select {
	case <-done:
		panic("Cond.Signal woke more than one waiter")
	default:
	}

	mu.Lock()
	cond.Broadcast()
	mu.Unlock()
	<-done
}

func testSyncHelpers() {
	var once sync.Once
	var wg sync.WaitGroup
	count := 0
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			once.Do(func() {
				count++
			})
			wg.Done()
		}()
	}
	wg.Wait()
	if count != 1 {
		panic("sync.Once ran more than once")
	}

	var pool sync.Pool
	pool.Put("pooled")
	if value := pool.Get(); value != "pooled" {
		panic("sync.Pool value mismatch")
	}

	var value atomic.Value
	value.Store("before")
	if old := value.Swap("after"); old != "before" || value.Load() != "after" {
		panic("atomic.Value swap mismatch")
	}
}

func testSyncMap() {
	var m sync.Map
	if _, loaded := m.LoadOrStore("key", 1); loaded {
		panic("sync.Map unexpectedly loaded a missing key")
	}
	if value, loaded := m.Load("key"); !loaded || value != 1 {
		panic("sync.Map load mismatch")
	}
	if !m.CompareAndSwap("key", 1, 2) {
		panic("sync.Map compare-and-swap failed")
	}
	if previous, loaded := m.Swap("key", 3); !loaded || previous != 2 {
		panic("sync.Map swap mismatch")
	}
	count := 0
	m.Range(func(key, value any) bool {
		if key != "key" || value != 3 {
			panic("sync.Map range mismatch")
		}
		count++
		return true
	})
	if count != 1 {
		panic("sync.Map range count mismatch")
	}
	if !m.CompareAndDelete("key", 3) {
		panic("sync.Map compare-and-delete failed")
	}
	if _, loaded := m.LoadAndDelete("key"); loaded {
		panic("sync.Map retained a deleted key")
	}
	m.Store("clear", 4)
	m.Clear()
	if _, loaded := m.Load("clear"); loaded {
		panic("sync.Map clear failed")
	}
}

func testGoroutineLifecycle() {
	const count = 5000
	for i := 1; i <= count; i++ {
		want := i
		go func() {
			lifecycle = want
		}()
		for lifecycle != want {
			runtime.Gosched()
		}
	}
}

func testParkedMainDeadlock() {
	go func() {}()
	parkForTesting()
	panic("park returned after scheduler deadlock")
}
