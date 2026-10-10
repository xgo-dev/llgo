package main

import (
	"runtime"
	"strings"

	"github.com/xgo-dev/llgo/cl/_testlto/caller_inline_entry/arith"
)

//go:noinline
func capture(pcs *[16]uintptr, value *int) int {
	// Cross-package LTO copies AddSeven's entry record into this function.
	// That record must not replace capture's identity in CallersFrames.
	*value = arith.AddSeven(*value)
	return runtime.Callers(1, pcs[:])
}

func main() {
	var pcs [16]uintptr
	value := 5
	n := capture(&pcs, &value)
	if value != 12 {
		panic("inlined computation returned the wrong value")
	}
	frames := runtime.CallersFrames(pcs[:n])
	found := 0
	for {
		frame, more := frames.Next()
		if strings.HasSuffix(frame.Function, ".capture") {
			found++
			if !strings.HasSuffix(frame.File, "/in.go") || frame.Line <= 0 {
				panic("capture frame lost its source location")
			}
		}
		if !more {
			break
		}
	}
	if found != 1 {
		panic("capture frame lost its physical function identity")
	}
}
