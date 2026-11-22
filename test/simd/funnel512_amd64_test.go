//go:build goexperiment.simd && amd64

package simd_test

import (
	"math/bits"
	"runtime"
	"simd/archsimd"
	"testing"
)

func TestSIMD512Funnel(t *testing.T) {
	if runtime.Compiler != "llgo" && !archsimd.X86.AVX512VBMI2() {
		t.Skip("requires AVX512VBMI2 in gc")
	}
	var a, b, c, out [8]uint32
	c = [8]uint32{0, 1, 31, 32, 33, 63, 64, ^uint32(0)}
	for i := range a {
		a[i] = 0x87654321 + uint32(i)
		b[i] = 0x12345678 - uint32(i)
	}
	x, y, count := archsimd.LoadUint32x8Array(&a), archsimd.LoadUint32x8Array(&b), archsimd.LoadUint32x8Array(&c)
	x.ShiftLeftConcatMod32(y, count).StoreArray(&out)
	for i, v := range out {
		k := c[i] & 31
		if want := a[i]<<k | b[i]>>(32-k); v != want {
			t.Fatalf("left lane %d = %#x, want %#x", i, v, want)
		}
	}
	for _, k := range c {
		x.ShiftAllRightConcatMod32(y, uint64(k)).StoreArray(&out)
		for i, v := range out {
			n := k & 31
			if want := a[i]>>n | b[i]<<(32-n); v != want {
				t.Fatalf("right count %d lane %d = %#x, want %#x", k, i, v, want)
			}
		}
	}
	x.RotateLeft(count).StoreArray(&out)
	for i, v := range out {
		if want := bits.RotateLeft32(a[i], int(c[i])); v != want {
			t.Fatalf("rotate lane %d = %#x, want %#x", i, v, want)
		}
	}
}

func TestSIMDFunnelComplements(t *testing.T) {
	if runtime.Compiler != "llgo" && !archsimd.X86.AVX512VBMI2() {
		t.Skip("requires AVX512VBMI2 in gc")
	}
	a, b := [8]uint32{}, [8]uint32{}
	counts := [8]uint32{0, 1, 15, 31, 32, 33, 64, ^uint32(0)}
	for i := range a {
		a[i] = 0xfedcba98 ^ uint32(i*11)
		b[i] = 0x1234567 + uint32(i*13)
	}
	x, y, c := archsimd.LoadUint32x8Array(&a), archsimd.LoadUint32x8Array(&b), archsimd.LoadUint32x8Array(&counts)
	var right, rotate, left [8]uint32
	x.ShiftRightConcatMod32(y, c).StoreArray(&right)
	x.RotateRight(c).StoreArray(&rotate)
	for i, k := range counts {
		n := k & 31
		if right[i] != a[i]>>n|b[i]<<(32-n) || rotate[i] != bits.RotateLeft32(a[i], -int(n)) {
			t.Fatalf("count %d lane %d: right %#x rotate %#x", k, i, right[i], rotate[i])
		}
	}
	for _, k := range counts {
		x.ShiftAllLeftConcatMod32(y, uint64(k)).StoreArray(&left)
		n := k & 31
		for i, v := range left {
			if v != a[i]<<n|b[i]>>(32-n) {
				t.Fatalf("left shift %d lane %d: %#x", k, i, v)
			}
		}
	}
}
