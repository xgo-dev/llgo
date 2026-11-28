//go:build goexperiment.simd && amd64

package simd_test

import (
	"os"
	"runtime"
	"simd/archsimd"
	"strings"
	"testing"
)

type outlineTrace struct {
	pcs  [3]uintptr
	skip [1]uintptr
	line int
}

//go:noinline
func fmvOutlineCallback(v archsimd.Int32x8, trace *outlineTrace) archsimd.Int32x8 {
	runtime.Callers(1, trace.pcs[:])
	runtime.Callers(2, trace.skip[:])
	return v
}

//go:noinline
func fmvOutlineCalls(v *archsimd.Int32x8, traces *[2]outlineTrace, cb func(archsimd.Int32x8, *outlineTrace) archsimd.Int32x8) {
	_, _, traces[0].line, _ = runtime.Caller(0)
	*v = cb(*v, &traces[0])
	_, _, traces[1].line, _ = runtime.Caller(0)
	*v = cb(*v, &traces[1])
}

func TestSIMDFMVOutlineCallers(t *testing.T) {
	// The masked pass is requested only after the runner has verified that
	// the host can execute the fixed YMM argument and return convention.
	if os.Getenv("LLGO_TEST_MASKED_YMM") == "1" {
		if archsimd.X86.AVX() {
			t.Fatal("masked test requires the AVX query to be false")
		}
	} else if !archsimd.X86.AVX() {
		t.Skip("requires YMM register support")
	}
	input := [8]int32{1, -2, 3, -4, 5, -6, 7, -8}
	v := archsimd.LoadInt32x8Array(&input)
	var traces [2]outlineTrace
	fmvOutlineCalls(&v, &traces, fmvOutlineCallback)
	var output [8]int32
	v.StoreArray(&output)
	if output != input {
		t.Fatalf("callback returned %v, want %v", output, input)
	}
	for i, trace := range traces {
		frames := runtime.CallersFrames(trace.pcs[:])
		for j, name := range []string{".fmvOutlineCallback", ".fmvOutlineCalls", ".TestSIMDFMVOutlineCallers"} {
			frame, more := frames.Next()
			if !strings.HasSuffix(frame.Function, name) || more != (j < 2) {
				t.Fatalf("call %d frame %d = %+v (more %t), want %s", i, j, frame, more, name)
			}
			if j == 1 && (!strings.HasSuffix(frame.File, "fmv_outline_amd64_test.go") || frame.Line != trace.line+1) {
				t.Fatalf("call %d at %s:%d, want line %d", i, frame.File, frame.Line, trace.line+1)
			}
		}
		if trace.skip[0] != trace.pcs[1] {
			t.Fatalf("call %d: Callers(2) = %#x, want caller PC %#x", i, trace.skip[0], trace.pcs[1])
		}
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected a nil dereference in the callback")
		}
		var pcs [32]uintptr
		n := runtime.Callers(1, pcs[:])
		frames := runtime.CallersFrames(pcs[:n])
		for {
			f, more := frames.Next()
			if strings.HasSuffix(f.Function, ".fmvOutlineFault") {
				caller, _ := frames.Next()
				if !strings.HasSuffix(caller.Function, ".fmvOutlineCalls") || caller.Line != traces[0].line+1 {
					t.Fatalf("recovered caller = %+v, want fmvOutlineCalls line %d", caller, traces[0].line+1)
				}
				return
			}
			if !more {
				t.Fatal("recovered stack lost the faulting callback")
			}
		}
	}()
	fmvOutlineCalls(&v, &traces, fmvOutlineFault)
}

var outlineFaultPointer *struct{ c chan int }
var outlineFaultSink chan int

//go:noinline
func fmvOutlineFault(v archsimd.Int32x8, trace *outlineTrace) archsimd.Int32x8 {
	outlineFaultSink = outlineFaultPointer.c
	return v
}
