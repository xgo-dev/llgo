package gotest

import (
	"runtime"
	"strings"
	"testing"
)

type callerStackPeano *callerStackPeano

var callerStackBottomPC uintptr

// Keep this below the smallest fixed Fiber stack budget used by the wasm
// profiles. It is still deep enough to catch per-frame aggregate temporaries
// in caller instrumentation, while remaining valid on Memory32's 128 KiB
// stack as well as Memory64's larger default.
const callerStackDepth = 2048

//go:noinline
func makeCallerStackPeano(n int, suspend bool) *callerStackPeano {
	if n == 0 {
		captureCallerStackBottom(suspend)
		return nil
	}
	p := callerStackPeano(makeCallerStackPeano(n-1, suspend))
	return &p
}

//go:noinline
func captureCallerStackBottom(suspend bool) {
	if suspend {
		// Force Asyncify to preserve the complete deep call chain, even when
		// the test is run alone and no other goroutine happens to be runnable.
		runtime.Gosched()
		runtime.GC()
	}
	callerStackBottomPC, _, _, _ = runtime.Caller(0)
}

func TestCallerInstrumentationDeepRecursion(t *testing.T) {
	for _, suspend := range []bool{false, true} {
		name := "normal"
		if suspend {
			name = "suspended"
		}
		t.Run(name, func(t *testing.T) {
			callerStackBottomPC = 0
			p := makeCallerStackPeano(callerStackDepth, suspend)
			depth := 0
			for p != nil {
				depth++
				p = *p
			}
			if depth != callerStackDepth {
				t.Fatalf("recursive pointer depth = %d, want %d", depth, callerStackDepth)
			}
			fn := runtime.FuncForPC(callerStackBottomPC)
			if fn == nil || !strings.HasSuffix(fn.Name(), ".captureCallerStackBottom") {
				t.Fatalf("deepest caller = %v at %#x", fn, callerStackBottomPC)
			}
		})
	}
}
