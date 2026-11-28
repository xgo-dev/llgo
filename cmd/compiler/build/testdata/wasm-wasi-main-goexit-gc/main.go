package main

import (
	"runtime"
	_ "unsafe"
)

//go:linkname usleep C.usleep
func usleep(useconds uint32) int32

//go:linkname gStateForTesting github.com/xgo-dev/llgo/runtime/internal/runtime.GStateForTesting
func gStateForTesting() (count uint64, mainExited bool)

func init() {
	go func() {
		for {
			_, mainExited := gStateForTesting()
			if mainExited {
				break
			}
			usleep(1000)
		}
		// Allow the initial pthread to enter its Goexit parking loop.
		usleep(20_000)
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		runtime.GC()
		runtime.ReadMemStats(&after)
		if after.NumGC > before.NumGC {
			println("wasi goexit gc collected")
		} else {
			println("wasi goexit gc skipped")
		}
	}()
	runtime.Goexit()
}

func main() {}
