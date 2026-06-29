package gotest

import (
	"os"
	"path/filepath"
	"testing"
)

const runtimePanicCallerLineProbe = `package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

var idx = 9
var nilStruct *struct {
	c   chan int
	val int
}
var sink int

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func fileOK(file string) bool {
	return strings.HasSuffix(file, "main.go") || strings.HasSuffix(file, "panic_lines.go")
}

func main() {
	explicitPanicCaller()
	nilFieldCallerLine()
	boundsFramesLine()
	deferReturnCallerLine()
}

func explicitPanicCaller() {
	const wantLine = 110
	defer func() {
		if recover() == nil {
			fail("explicit panic did not panic")
		}
		_, file, line, ok := runtime.Caller(2)
		if !ok || !fileOK(file) || line != wantLine {
			fail("runtime.Caller(2) = %s:%d ok=%v, want line %d", file, line, ok, wantLine)
		}
	}()
//line panic_lines.go:110
	panic("boom")
}

func nilFieldCallerLine() {
	const wantLine = 311
	defer func() {
		if recover() == nil {
			fail("nil field did not panic")
		}
		for i := 0; ; i++ {
			pc, file, line, ok := runtime.Caller(i)
			if !ok {
				fail("runtime.Caller could not find nilFieldCallerLine")
			}
			fn := runtime.FuncForPC(pc)
			name := ""
			if fn != nil {
				name = fn.Name()
			}
			if !strings.HasSuffix(name, ".nilFieldCallerLine") {
				continue
			}
			if !fileOK(file) || line != wantLine {
				fail("nil field frame = %s:%d %s, want line %d", file, line, name, wantLine)
			}
			return
		}
	}()
//line panic_lines.go:310
	select {
	case <-nilStruct.c:
	default:
	}
}

func boundsFramesLine() {
	const wantLine = 999999
	defer func() {
		if recover() == nil {
			fail("bounds check did not panic")
		}
		var pcs [16]uintptr
		n := runtime.Callers(1, pcs[:])
		frames := runtime.CallersFrames(pcs[:n])
		for {
			frame, more := frames.Next()
			if strings.HasSuffix(frame.Function, ".boundsFramesLine") {
				if !fileOK(frame.File) || frame.Line != wantLine {
					fail("bounds frame = %s:%d %s, want line %d", frame.File, frame.Line, frame.Function, wantLine)
				}
				return
			}
			if !more {
				break
			}
		}
		fail("CallersFrames could not find boundsFramesLine")
	}()
	var a [1]int
//line panic_lines.go:999999
	sink = a[idx]
}

//line main.go:500
func deferReturnCallerLine() {
	const wantLine = 507
	got := 0
	func() {
		defer func() {
			_, _, got, _ = runtime.Caller(1)
		}()
	}()
	if got != wantLine {
		fail("defer runtime.Caller(1) line = %d, want %d", got, wantLine)
	}
}
`

func TestRuntimePanicCallerLineAttribution(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte(runtimePanicCallerLineProbe), 0644); err != nil {
		t.Fatal(err)
	}
	root := findLLGoRoot(t)
	t.Setenv("LLGO_ROOT", root)
	runGoCmd(t, root, "run", "./cmd/llgo", "run", file)
}
