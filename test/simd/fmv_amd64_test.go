//go:build goexperiment.simd && amd64

package simd_test

import (
	"runtime"
	"simd/archsimd"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/test/simd/internal/vectorcall"
)

//go:noinline
func fmvGuarded(x archsimd.Float32x4) archsimd.Float32x4 {
	if archsimd.X86.AVX2() {
		x = vectorcall.Echo(x, false)
		// GODEBUG may disable FMA independently of AVX2. Instruction support
		// must never replace the observable result of this separate query.
		if archsimd.X86.FMA() {
			x = x.Add(archsimd.BroadcastFloat32x4(3))
		} else {
			x = x.Sub(archsimd.BroadcastFloat32x4(3))
		}
		if !archsimd.X86.AVX() {
			x = x.Add(archsimd.BroadcastFloat32x4(11))
		}
		return x
	}
	return x.Sub(archsimd.BroadcastFloat32x4(7))
}

var fmvInitial = fmvGuarded(archsimd.BroadcastFloat32x4(10))

func TestSIMDFMVDispatch(t *testing.T) {
	want := float32(3)
	if archsimd.X86.AVX2() {
		want = 9
		if archsimd.X86.FMA() {
			want = 15
		}
		if !archsimd.X86.AVX() {
			want += 11
		}
	}
	t.Logf("AVX=%v AVX2=%v FMA=%v", archsimd.X86.AVX(), archsimd.X86.AVX2(), archsimd.X86.FMA())
	fn := fmvGuarded
	for _, value := range []archsimd.Float32x4{fmvInitial, fmvGuarded(archsimd.BroadcastFloat32x4(10)), fn(archsimd.BroadcastFloat32x4(10))} {
		var lanes [4]float32
		value.StoreArray(&lanes)
		for _, got := range lanes {
			if got != want {
				t.Fatalf("guarded result %v, want %g", lanes, want)
			}
		}
	}
}

//go:noinline
func fmvTrace(x archsimd.Float32x4, pcs *[20]uintptr, n *int) archsimd.Float32x4 {
	if archsimd.X86.AVX2() {
		*n = runtime.Callers(0, pcs[:])
		return vectorcall.Echo(x, false)
	}
	*n = runtime.Callers(0, pcs[:])
	return x
}

func TestSIMDFMVTraceback(t *testing.T) {
	var pcs [20]uintptr
	var n int
	fmvTrace(archsimd.BroadcastFloat32x4(1), &pcs, &n)
	frames := runtime.CallersFrames(pcs[:n])
	count := 0
	var trace []runtime.Frame
	for {
		frame, more := frames.Next()
		trace = append(trace, frame)
		if strings.Contains(frame.Function, "__llgo_fmv") {
			t.Fatalf("compiler variant leaked into traceback: %+v", frame)
		}
		if strings.HasSuffix(frame.Function, ".fmvTrace") {
			count++
			if !strings.HasSuffix(frame.File, "fmv_amd64_test.go") || frame.Line == 0 {
				t.Fatalf("missing source location: %+v", frame)
			}
		}
		if !more {
			break
		}
	}
	if count != 1 {
		t.Fatalf("expected one fmvTrace frame, got %d; frames: %+v", count, trace)
	}
}
