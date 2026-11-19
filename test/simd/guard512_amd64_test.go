//go:build goexperiment.simd && amd64

package simd_test

import (
	"runtime"
	"simd/archsimd"
	"testing"
)

//go:noinline
func guardedSIMD512(x, y archsimd.Float32x16) (archsimd.Float32x16, int) {
	if archsimd.X86.AVX512() {
		v := x.Add(y)
		if archsimd.X86.FMA() {
			return v.MulAdd(y, x), 3
		}
		return v, 2
	}
	return x, 1
}

func TestSIMD512Guarded(t *testing.T) {
	if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
		t.Skip("gc requires AVX512 to pass a 512-bit value")
	}
	// Pass only zero-initialized vectors before testing the hardware predicate.
	var zero archsimd.Float32x16
	_, path := guardedSIMD512(zero, zero)
	want := 1
	if archsimd.X86.AVX512() {
		want = 2
		if archsimd.X86.FMA() {
			want = 3
		}
	}
	if path != want {
		t.Fatalf("guarded path = %d, want %d", path, want)
	}
	if !archsimd.X86.AVX512() {
		return
	}
	x, y := archsimd.BroadcastFloat32x16(2), archsimd.BroadcastFloat32x16(4)
	out, _ := guardedSIMD512(x, y)
	var a [16]float32
	out.StoreArray(&a)
	value := float32(6)
	if want == 3 {
		value = 26
	}
	for i, v := range a {
		if v != value {
			t.Fatalf("lane %d = %v, want %v", i, v, value)
		}
	}
}
