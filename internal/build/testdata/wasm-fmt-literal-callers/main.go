package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
)

var dynamicFormat = "literal%%"
var stdoutFormat = "%v\n"

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

type redirectStdout struct{ sawFprintf *bool }

//go:noinline
func (s redirectStdout) Format(w fmt.State, _ rune) {
	os.Stdout = nil
	var pcs [32]uintptr
	n := runtime.Callers(0, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if frame.Function == "fmt.Fprintf" {
			*s.sawFprintf = true
		}
		if !more {
			break
		}
	}
	w.Write([]byte("stdout captured before formatting"))
}

func checkStdout() {
	stdout := os.Stdout
	defer func() { os.Stdout = stdout }()

	// The logical frame may be unavailable under a particular Wasm build
	// policy. Compare the optimized entry with the regular formatting path.
	var directFrame, stdoutFrame bool
	fmt.Fprintf(io.Discard, stdoutFormat, redirectStdout{&directFrame})
	os.Stdout = stdout
	n, err := fmt.Printf(stdoutFormat, redirectStdout{&stdoutFrame})
	if n != len("stdout captured before formatting\n") || err != nil || os.Stdout != nil {
		panic("Printf changed stdout evaluation or write result")
	}
	if stdoutFrame != directFrame {
		panic("Printf changed Fprintf frame visibility")
	}
	if n, err := fmt.Print("nil stdout"); n != 0 || err != os.ErrInvalid {
		panic("Print lost the nil stdout write error")
	}
	if n, err := fmt.Println("nil stdout"); n != 0 || err != os.ErrInvalid {
		panic("Println lost the nil stdout write error")
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
	checkStdout()
	println("literal format caller parity passed")
}
