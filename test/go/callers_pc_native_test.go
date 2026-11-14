//go:build !wasm

package gotest

import (
	"runtime"
	"strconv"
	"strings"
	"testing"
)

//go:noinline
func captureFramesReturnPC(pcs []uintptr) int {
	return runtime.Callers(0, pcs)
}

//go:noinline
func captureFramesTraceback(pcs []uintptr, stack []byte) (int, int) {
	n := runtime.Callers(0, pcs)
	return n, runtime.Stack(stack, false)
}

func TestCallersFramesTracebackOffset(t *testing.T) {
	var pcs [32]uintptr
	var stack [8192]byte
	n, size := captureFramesTraceback(pcs[:], stack[:])
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if strings.HasSuffix(frame.Function, ".TestCallersFramesTracebackOffset") {
			lines := strings.Split(string(stack[:size]), "\n")
			for i, line := range lines {
				if !strings.HasPrefix(line, frame.Function+"(") || i+1 >= len(lines) {
					continue
				}
				source := lines[i+1]
				if pos := strings.LastIndex(source, "+0x"); pos >= 0 {
					offset, err := strconv.ParseUint(strings.TrimSpace(source[pos+3:]), 16, 64)
					want := uint64(frame.PC + 1 - frame.Entry)
					if err != nil || offset != want {
						t.Fatalf("traceback offset = %#x (%v), want return-PC offset %#x:\n%s", offset, err, want, stack[:size])
					}
					return
				}
			}
			t.Fatalf("traceback is missing the current frame's offset:\n%s", stack[:size])
		}
		if !more {
			t.Fatal("current caller frame is missing")
		}
	}
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
