package main

import (
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

const LLGoFiles = "_wrap/block.c"

type payload struct {
	value int
	next  *payload
}

var stop atomic.Bool

//go:linkname blockInC C.llgo_gc_block_in_c
func blockInC(unsafe.Pointer)

//go:linkname cBlocked C.llgo_gc_c_blocked
func cBlocked() int32

//go:linkname releaseC C.llgo_gc_release_c
func releaseC()

//go:linkname gStateForTesting github.com/xgo-dev/llgo/runtime/internal/runtime.GStateForTesting
func gStateForTesting() (count uint64, mainExited bool)

//go:linkname yieldC C.sched_yield
func yieldC() int32

//go:linkname worldRegistered C.llgo_wasi_gc_registered
func worldRegistered() int32

func fail(message string) {
	println(message)
	os.Exit(2)
}

func main() {
	baseline, _ := gStateForTesting()
	const workers = 3
	var wg sync.WaitGroup
	ready := make(chan *payload, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			inner := &payload{value: index + 100}
			head := &payload{value: index + 200, next: inner}
			ready <- head
			for !stop.Load() {
				garbage := make([]byte, 2048)
				garbage[0] = byte(index)
				if head.next.value != index+100 {
					fail("concurrent root was lost")
				}
				runtime.Gosched()
				yieldC()
			}
		}(i)
	}
	heads := make([]*payload, workers)
	for i := range heads {
		heads[i] = <-ready
	}
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < 6; i++ {
		runtime.GC()
		for _, head := range heads {
			if head == nil || head.next == nil || head.value != head.next.value+100 {
				fail("cross-thread pointer handoff failed")
			}
		}
	}
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	if after.NumGC <= before.NumGC {
		fail("concurrent collection did not run")
	}
	stop.Store(true)
	wg.Wait()
	waitForBaseline(baseline)

	// A parked channel receiver still owns a pthread in this backend. It
	// must acknowledge a collection without requiring a sender to wake it.
	parked := make(chan struct{})
	parkedReady := make(chan struct{})
	parkedDone := make(chan struct{})
	go func() {
		close(parkedReady)
		<-parked
		close(parkedDone)
	}()
	<-parkedReady
	waitForRegistered(2)
	for i := 0; i < 1000; i++ {
		yieldC()
	}
	runtime.ReadMemStats(&before)
	runtime.GC()
	runtime.ReadMemStats(&after)
	if after.NumGC <= before.NumGC {
		fail("collection stalled while channel receiver was parked")
	}
	close(parked)
	<-parkedDone
	waitForBaseline(baseline)

	blocked := make(chan *payload, 1)
	unblocked := make(chan struct{})
	go func() {
		value := &payload{value: 0x1234}
		blocked <- value
		blockInC(unsafe.Pointer(value))
		if value.value != 0x1234 {
			fail("blocked C call lost its Go pointer")
		}
		close(unblocked)
	}()
	value := <-blocked
	for cBlocked() == 0 {
		yieldC()
	}
	runtime.ReadMemStats(&before)
	runtime.GC()
	runtime.ReadMemStats(&after)
	if after.NumGC != before.NumGC || value.value != 0x1234 {
		fail("collector swept while C code was blocked")
	}
	releaseC()
	<-unblocked
	waitForBaseline(baseline)
	runtime.GC()
	runtime.ReadMemStats(&after)
	if after.NumGC <= before.NumGC {
		fail("collection did not resume after C returned")
	}

	// The timer service owns a persistent pthread. Both a distant deadline
	// and an empty timer heap must let it acknowledge collection requests.
	timer := time.NewTimer(time.Hour)
	waitForRegistered(2)
	runtime.ReadMemStats(&before)
	runtime.GC()
	runtime.ReadMemStats(&after)
	if after.NumGC <= before.NumGC {
		fail("collection stalled while timer was waiting")
	}
	timer.Stop()
	runtime.ReadMemStats(&before)
	runtime.GC()
	runtime.ReadMemStats(&after)
	if after.NumGC <= before.NumGC {
		fail("collection stalled after timer stopped")
	}
	if !recoverPayload(true) {
		fail("main-thread panic/recover failed")
	}
	recovered := make(chan bool, 1)
	go func() { recovered <- recoverPayload(false) }()
	if !<-recovered {
		fail("worker panic/recover failed")
	}
	// Live Go allocations must grow beyond the first libc arena without
	// overlapping the pthread stacks and TLS allocated between arenas.
	retained := make([]byte, 33<<20)
	retained[0] = 0x12
	retained[len(retained)-1] = 0x34
	runtime.GC()
	if retained[0] != 0x12 || retained[len(retained)-1] != 0x34 {
		fail("live allocation was lost across arena growth")
	}
	if value.value != 0x1234 {
		fail("earlier arena was lost after growth")
	}
	runtime.ReadMemStats(&after)
	if after.HeapSys <= 32<<20 {
		fail("threaded GC heap did not grow")
	}
	runtime.KeepAlive(retained)
	waitForRegistered(2)
	println("wasi threaded gc ok")
}

// LLVM lowers this recover path through legacy Wasm EH. It must keep working
// on the WAMR build used by CI, including a collection in the active defer.
func recoverPayload(collect bool) (ok bool) {
	value := &payload{value: 0x5678}
	defer func() {
		recovered, valid := recover().(*payload)
		if collect {
			runtime.GC()
		}
		ok = valid && recovered != nil && recovered.value == value.value
	}()
	panic(value)
}

func waitForBaseline(baseline uint64) {
	for i := 0; i < 100000; i++ {
		count, _ := gStateForTesting()
		if count == baseline && worldRegistered() == 1 {
			return
		}
		yieldC()
	}
	fail("WASI pthreads did not retire")
}

func waitForRegistered(expected int32) {
	for i := 0; i < 100000; i++ {
		if worldRegistered() == expected {
			return
		}
		yieldC()
	}
	fail("WASI timer pthread did not start")
}
