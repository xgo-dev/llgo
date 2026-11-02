package main

import (
	"runtime"
	_ "unsafe"
)

const LLGoFiles = "_wrap/probe.cpp"

//go:linkname cppProbe C.llgo_debug_cpp_probe
func cppProbe(int32) int32

//go:linkname growMemory C.llgo_debug_grow_memory
func growMemory() int32

//go:noinline
func debugWaiter(depth int, ready chan<- bool, release <-chan bool, done chan<- bool) {
	local := [4]int{depth, depth + 11, depth + 22, depth + 33}
	index := depth & 3
	local[index] += 100
	if depth != 0 {
		debugWaiter(depth-1, ready, release, done)
		if local[index] != depth+index*11+100 {
			panic("parked local changed")
		}
		return
	}
	ready <- true
	<-release
	if local[index] != depth+index*11+100 {
		panic("parked local changed")
	}
	done <- true
}

//go:noinline
func debugRecoveredWaiter(ready chan<- bool, release <-chan bool, done chan<- bool) {
	defer func() {
		ready <- true
		<-release
		done <- true
	}()
	defer func() { recover() }()
	debugPanicLeaf()
}

//go:noinline
func debugPanicLeaf() { panic("parked panic snapshot") }

func main() {
	prepareDebugViews()
	ready, release, done := make(chan bool, 2), make(chan bool), make(chan bool, 2)
	spawnDebugG(func() { debugWaiter(3, ready, release, done) })
	spawnDebugG(func() { debugWaiter(4, ready, release, done) })
	<-ready
	<-ready
	spawnDebugG(func() { debugRecoveredWaiter(ready, release, done) })
	<-ready
	if growMemory() == 0 {
		panic("debug fixture memory did not grow")
	}
	runtime.GC()
	if cppProbe(14) != 21 {
		panic("debug probe returned the wrong value")
	}
	checkDebugViews()
	close(release)
	<-done
	<-done
	<-done
	// Allow the senders to return and retire their runtime contexts before
	// the second debugger snapshot checks that their records were removed.
	for i := 0; i < 32; i++ {
		runtime.Gosched()
	}
	runtime.GC()
	if cppProbe(14) != 21 {
		panic("debug cleanup probe returned the wrong value")
	}
	println("wasm debug ok")
}
