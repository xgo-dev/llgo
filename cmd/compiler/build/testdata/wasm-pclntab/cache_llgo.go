//go:build llgo

package main

import (
	"runtime"
	"strings"
	"unsafe"
)

//go:noinline
func checkCallerCache() {
	var firstPC uintptr
	var firstFile string
	for i := range 4 {
		pc, file, _, ok := runtime.Caller(0)
		if !ok || file == "" || file == "???" {
			panic("missing cached caller source")
		}
		if i == 0 {
			firstPC, firstFile = pc, file
		} else if pc != firstPC || unsafe.StringData(file) != unsafe.StringData(firstFile) {
			panic("caller source cache was not reused")
		}
	}
}

//go:noinline
func checkCallersCache() {
	var first runtime.Frame
	for i := range 4 {
		var pcs [16]uintptr
		n := runtime.Callers(0, pcs[:])
		frames := runtime.CallersFrames(pcs[:n])
		found := false
		for {
			frame, more := frames.Next()
			if strings.HasSuffix(frame.Function, ".checkCallersCache") {
				found = true
				if i == 0 {
					first = frame
				} else if frame.PC != first.PC || unsafe.StringData(frame.File) != unsafe.StringData(first.File) ||
					unsafe.StringData(frame.Function) != unsafe.StringData(first.Function) {
					panic("callers source cache was not reused")
				}
			}
			if !more {
				break
			}
		}
		if !found {
			panic("missing cached callers frame")
		}
	}
}
