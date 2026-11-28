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

//go:noinline
func savedGuard(dst, src *[16]int32, enabled bool) bool {
	ready := archsimd.X86.AVX512() && enabled
	if ready {
		doubleSIMD512Kernel(dst, src)
	}
	return ready
}

func TestSIMDSavedGuard(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		in, dst := [16]int32{3, 5}, [16]int32{7}
		want := dst
		take := enabled && archsimd.X86.AVX512()
		if take {
			want = [16]int32{6, 10}
		}
		if got := savedGuard(&dst, &in, enabled); got != take || dst != want {
			t.Fatalf("enabled=%t: path %t, result %v; want %t, %v", enabled, got, dst, take, want)
		}
	}
}

//go:noinline
func allowWide(count *int, enabled bool) bool {
	*count++
	return enabled && archsimd.X86.AVX512()
}

//go:noinline
func viaPredicateHelper(dst, src *[16]int32, count *int, enabled bool) {
	if allowWide(count, enabled) {
		doubleSIMD512Kernel(dst, src)
	}
}

func TestSIMDPredicateHelper(t *testing.T) {
	var count int
	for i, enabled := range []bool{true, false, true} {
		in, dst := [16]int32{-2, 9}, [16]int32{11}
		want := dst
		if enabled && archsimd.X86.AVX512() {
			want = [16]int32{-4, 18}
		}
		viaPredicateHelper(&dst, &in, &count, enabled)
		if count != i+1 || dst != want {
			t.Fatalf("enabled=%t: count %d, dst %v; want %d, %v", enabled, count, dst, i+1, want)
		}
	}
}
