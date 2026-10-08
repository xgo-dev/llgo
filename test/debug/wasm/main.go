package main

import (
	"runtime"
	_ "unsafe"
)

const LLGoFiles = "_wrap/probe.cpp"

//go:linkname cppProbe C.llgo_debug_cpp_probe
func cppProbe(int32) int32

//go:noinline
func debugWaiter(depth int, ready chan<- bool, release <-chan bool, done chan<- bool) {
	if depth != 0 {
		debugWaiter(depth-1, ready, release, done)
		return
	}
	ready <- true
	<-release
	done <- true
}

func main() {
	ready, release, done := make(chan bool, 2), make(chan bool), make(chan bool, 2)
	spawnDebugG(func() { debugWaiter(3, ready, release, done) })
	spawnDebugG(func() { debugWaiter(4, ready, release, done) })
	<-ready
	<-ready
	runtime.GC()
	if cppProbe(14) != 21 {
		panic("debug probe returned the wrong value")
	}
	close(release)
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
