package main

import (
	"fmt"
	"runtime"
)

type writer struct{ found bool }

func (w *writer) Write(p []byte) (int, error) {
	var pcs [16]uintptr
	n := runtime.Callers(0, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if frame.Function == "fmt.llgoFprintfLiteral" {
			panic("literal helper leaked into debugger frames")
		}
		w.found = w.found || frame.Function == "fmt.Fprintf"
		if !more {
			break
		}
	}
	return len(p), nil
}

func main() {
	var w writer
	fmt.Fprintf(&w, "literal%%")
	if !w.found {
		panic("fmt.Fprintf missing from debugger frames")
	}
	println("literal format debugger frames passed")
}
