//go:build goexperiment.simd && amd64

package simd_test

import (
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
	if !archsimd.X86.AVX512() {
		t.Skip("requires AVX512")
	}
	x, y := archsimd.BroadcastFloat32x16(2), archsimd.BroadcastFloat32x16(4)
	out, path := guardedSIMD512(x, y)
	want := 2
	if archsimd.X86.FMA() {
		want = 3
	}
	if path != want {
		t.Fatalf("guarded path = %d, want %d", path, want)
	}
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

//go:noinline
func doubleSIMD512Kernel(dst, src *[16]int32) {
	x := archsimd.LoadInt32x16Array(src)
	x.Add(x).StoreArray(dst)
}

//go:noinline
func doubleSIMD512Helper(dst, src *[16]int32) {
	doubleSIMD512Kernel(dst, src)
}

//go:noinline
func doubleSIMD512(dst, src *[16]int32) bool {
	if archsimd.X86.AVX512() {
		doubleSIMD512Helper(dst, src)
		return true
	}
	for i, v := range src {
		dst[i] = 2 * v
	}
	return false
}

func TestSIMD512ScalarSignatureHelpers(t *testing.T) {
	var input, out [16]int32
	for i := range input {
		input[i] = int32(i - 8)
	}
	if wide := doubleSIMD512(&out, &input); wide != archsimd.X86.AVX512() {
		t.Fatalf("guarded path %t, AVX512 %t", wide, archsimd.X86.AVX512())
	}
	for i, got := range out {
		if want := 2 * input[i]; got != want {
			t.Fatalf("lane %d = %d, want %d", i, got, want)
		}
	}
}

//go:noinline
func mixedSIMD512(dst, src *[16]int32) int {
	if archsimd.X86.AVX2() {
		if archsimd.X86.AVX512() {
			doubleSIMD512Helper(dst, src)
			return 2
		}
		return 1
	}
	return 0
}

func TestSIMD512MixedGuard(t *testing.T) {
	var src, dst, want [16]int32
	for i := range src {
		src[i] = int32(i - 9)
	}
	path := 0
	if archsimd.X86.AVX2() {
		path = 1
		if archsimd.X86.AVX512() {
			path = 2
			for i, x := range src {
				want[i] = x + x
			}
		}
	}
	for call := 0; call < 2; call++ {
		if got := mixedSIMD512(&dst, &src); got != path || dst != want {
			t.Fatalf("call %d: path %d, lanes %v; want path %d, lanes %v", call, got, dst, path, want)
		}
	}
}
