//go:build !wasm && !baremetal

package gotest

import (
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
)

// Keep every recursive activation observable, even with optimization enabled.
//
//go:noinline
func tracebackDepth(depth int, f func()) int {
	if depth == 0 {
		f()
		return 0
	}
	return 1 + tracebackDepth(depth-1, f)
}

func TestRuntimeStackDepth(t *testing.T) {
	// Exact 50/100-frame formatting boundaries are covered by the formatter's
	// unit tests. Here exercise shallow, boundary-adjacent and deep real stacks.
	for _, depth := range []int{0, 45, 70, 95, 100, 250, 5000} {
		t.Run(strconv.Itoa(depth), func(t *testing.T) {
			tracebackDepth(depth, func() {
				buf := make([]byte, 64<<10)
				n := runtime.Stack(buf, false)
				checkTracebackDepth(t, string(buf[:n]), depth)
				checkTracebackDepth(t, string(debug.Stack()), depth)
			})
		})
	}
}

func checkTracebackDepth(t *testing.T, stack string, depth int) {
	t.Helper()
	if !strings.Contains(stack, ".TestRuntimeStackDepth") ||
		!strings.Contains(stack, "testing.tRunner(") {
		t.Fatalf("missing innermost or outermost frame:\n%s", stack)
	}
	frames, elided, beforeElision := 0, 0, -1
	for _, line := range strings.Split(stack, "\n") {
		if strings.HasPrefix(line, "...") && strings.HasSuffix(line, " frames elided...") {
			beforeElision = frames
			count := strings.TrimSuffix(strings.TrimPrefix(line, "..."), " frames elided...")
			var err error
			elided, err = strconv.Atoi(count)
			if err != nil || elided <= 0 {
				t.Fatalf("invalid elision marker %q", line)
			}
		} else if strings.Contains(line, "(") && strings.HasSuffix(line, ")") {
			frames++
		}
	}
	if elided != 0 {
		if frames != 100 || beforeElision != 50 {
			t.Fatalf("frames=%d, frames before elision=%d; want 100 and 50:\n%s", frames, beforeElision, stack)
		}
	} else if frames > 100 {
		t.Fatalf("printed %d frames without elision:\n%s", frames, stack)
	}
	// The recursion itself has depth+1 activations. Additional frames depend
	// on the compiler's inlining and testing runtime.
	if frames+elided < depth+1 {
		t.Fatalf("only %d frames accounted for at depth %d:\n%s", frames+elided, depth, stack)
	}
	if depth < 80 && elided != 0 {
		t.Fatalf("shallow stack was elided:\n%s", stack)
	}
	if depth >= 100 && elided == 0 {
		t.Fatalf("deep stack was not elided:\n%s", stack)
	}
}

func TestRuntimeCallersDepth(t *testing.T) {
	for _, depth := range []int{70, 250, 5000} {
		t.Run(strconv.Itoa(depth), func(t *testing.T) {
			tracebackDepth(depth, func() {
				pcs := make([]uintptr, depth+32)
				n := runtime.Callers(0, pcs)
				frames := runtime.CallersFrames(pcs[:n])
				got := 0
				for {
					frame, more := frames.Next()
					if strings.HasSuffix(frame.Function, ".tracebackDepth") {
						got++
					}
					if !more {
						break
					}
				}
				if got != depth+1 {
					t.Fatalf("Callers returned %d recursive frames, want %d", got, depth+1)
				}
			})
		})
	}
}

func TestRuntimeStackShortBuffer(t *testing.T) {
	for _, size := range []int{0, 1, 10, 64} {
		for _, all := range []bool{false, true} {
			storage := make([]byte, size+128)
			for i := size; i < len(storage); i++ {
				storage[i] = 0xff
			}
			if n := runtime.Stack(storage[:size], all); n != size {
				t.Errorf("Stack with %d bytes returned %d", size, n)
			}
			for _, b := range storage[size:] {
				if b != 0xff {
					t.Fatal("Stack wrote beyond the supplied slice length")
				}
			}
		}
	}
}

//go:noinline
func tracebackPanicAtDepth(depth int) {
	if depth == 0 {
		panic("depth")
	}
	tracebackPanicAtDepth(depth - 1)
}

func TestRuntimeCallersDeepRecover(t *testing.T) {
	func() {
		defer func() {
			if recover() != "depth" {
				t.Fatal("missing panic")
			}
			// The recovering activation is beyond the old 128-PC live walk.
			tracebackDepth(250, func() {
				pcs := make([]uintptr, 1024)
				n := runtime.Callers(0, pcs)
				frames := runtime.CallersFrames(pcs[:n])
				live, panicked := 0, 0
				for {
					frame, more := frames.Next()
					if strings.HasSuffix(frame.Function, ".tracebackDepth") {
						live++
					}
					if strings.HasSuffix(frame.Function, ".tracebackPanicAtDepth") {
						panicked++
					}
					if !more {
						break
					}
				}
				if live != 251 || panicked != 251 {
					t.Fatalf("Callers in recover: live=%d panic=%d, want 251 each", live, panicked)
				}
			})
		}()
		tracebackPanicAtDepth(250)
	}()
	// Once the recovering activation returns, it must not retain or splice
	// the previous panic into an unrelated stack.
	if stack := string(debug.Stack()); strings.Contains(stack, ".tracebackPanicAtDepth(") {
		t.Fatalf("stale recovered panic in stack:\n%s", stack)
	}
}
