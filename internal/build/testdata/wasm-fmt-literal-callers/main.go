package main

import (
	"fmt"
	"runtime"
)

var dynamicFormat = "literal%%"

type writer struct{ frames []string }

//go:noinline
func (w *writer) Write(p []byte) (int, error) {
	var pcs [32]uintptr
	n := runtime.Callers(0, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if frame.Function == "fmt.llgoFprintfLiteral" {
			panic("literal helper changed Writer callers")
		}
		w.frames = append(w.frames, frame.Function)
		if !more {
			break
		}
	}
	if string(p) != "literal%" {
		panic("Writer received wrong text")
	}
	return len(p), nil
}

// Both calls share the same caller. The global format keeps the reference call
// out of literal specialization even though its runtime text is identical.
//
//go:noinline
func write(w *writer, dynamic bool) {
	var n int
	var err error
	if dynamic {
		n, err = fmt.Fprintf(w, dynamicFormat)
	} else {
		n, err = fmt.Fprintf(w, "literal%%")
	}
	if n != len("literal%") || err != nil {
		panic("Fprintf returned wrong result")
	}
}

func main() {
	var literal, reference writer
	write(&literal, false)
	write(&reference, true)
	if len(literal.frames) == 0 || len(literal.frames) != len(reference.frames) {
		panic("Writer caller count changed")
	}
	for i, name := range literal.frames {
		if name != reference.frames[i] {
			panic("Writer caller identity changed")
		}
	}
	println("literal format caller parity passed")
}
