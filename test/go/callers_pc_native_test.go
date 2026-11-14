//go:build !wasm

package gotest

import (
	"runtime"
	"strings"
	"testing"
)

//go:noinline
func captureFramesReturnPC(pcs []uintptr) int {
	return runtime.Callers(0, pcs)
}

func TestCallersFramesAdjustsReturnPC(t *testing.T) {
	var pcs [32]uintptr
	n := captureFramesReturnPC(pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if strings.HasSuffix(frame.Function, ".captureFramesReturnPC") {
			// Callers returns return addresses; Frame.PC identifies the call
			// instruction, as Caller does. The physical return PC is one later.
			for _, pc := range pcs[:n] {
				if frame.PC+1 == pc {
					return
				}
			}
			t.Fatalf("frame PC %#x is not an adjusted return PC: %x", frame.PC, pcs[:n])
		}
		if !more {
			t.Fatal("capture frame is missing")
		}
	}
}
