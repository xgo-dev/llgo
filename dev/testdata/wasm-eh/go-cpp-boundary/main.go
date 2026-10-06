package main

import (
	"runtime"
	"time"

	"github.com/xgo-dev/llgo/dev/testdata/wasm-eh/go-cpp-boundary/cpp"
)

func main() {
	if mode := cpp.Mode(); mode != 0 {
		cpp.CatchAndSuspend(mode - 1)
		panic("unsupported catch suspension returned")
	}
	status := cpp.Catch()
	if status != 7 {
		panic("C++ did not catch its exception")
	}
	// The foreign catch must have completed before Go suspends or collects.
	time.Sleep(time.Millisecond)
	runtime.GC()
	defer func() {
		time.Sleep(time.Millisecond)
		runtime.GC()
		if recover() != status {
			panic("Go did not recover the translated status")
		}
		println("go cpp boundary ok")
	}()
	// Translate the C ABI status in Go. Never unwind a C++ exception through Go.
	panic(status)
}

//export llgo_eh_suspend_callback
//go:noinline
func suspendCallback() {
	println("catch suspend entered")
	time.Sleep(time.Millisecond)
	println("unsupported catch suspension resumed")
}
