package gotest

import (
	"runtime"
	"strings"
	"testing"
)

type callerStackPeano *callerStackPeano

var callerStackBottomPC uintptr

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
			p := makeCallerStackPeano(4096, suspend)
			depth := 0
			for p != nil {
				depth++
				p = *p
			}
			if depth != 4096 {
				t.Fatalf("recursive pointer depth = %d, want 4096", depth)
			}
			fn := runtime.FuncForPC(callerStackBottomPC)
			if fn == nil || !strings.HasSuffix(fn.Name(), ".captureCallerStackBottom") {
				t.Fatalf("deepest caller = %v at %#x", fn, callerStackBottomPC)
			}
		})
	}
}
