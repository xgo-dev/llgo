package main

import (
	"os"
	"runtime"
	"runtime/debug"

	"github.com/xgo-dev/llgo/benchmark/memprofile/internal/runner"
)

func main() {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "rate0":
			runtime.MemProfileRate = 0
		case "rate256m":
			runtime.MemProfileRate = 256 << 20
		}
	}
	runtime.MemProfile(nil, false)
	debug.SetGCPercent(-1)
	println(runner.Run().Nanoseconds())
}
