package runtime

import llrt "github.com/goplus/llgo/runtime/internal/runtime"

func NumCPU() int {
	return int(c_maxprocs())
}

func Breakpoint() {
	c_debugtrap()
}

func Gosched() {
	llrt.SetMemProfileRate(MemProfileRate)
}
