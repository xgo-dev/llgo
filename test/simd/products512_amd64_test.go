//go:build goexperiment.simd && amd64

package simd_test

import (
	"runtime"
	"simd/archsimd"
	"testing"
)

func TestSIMD512Products(t *testing.T) {
	if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
		t.Skip("requires AVX512 in gc")
	}
	var a [64]uint8
	var b [64]int8
	for i := range a {
		a[i] = 255
		b[i] = 127
		if i%4 >= 2 {
			b[i] = -128
		}
	}
	x := archsimd.LoadUint8x64Array(&a)
	y := archsimd.LoadInt8x64Array(&b)
	var out [32]int16
	x.DotProductPairsSaturated(y).StoreArray(&out)
	for i, got := range out {
		want := int16(32767)
		if i%2 != 0 {
			want = -32768
		}
		if got != want {
			t.Fatalf("product pair %d = %d, want %d", i, got, want)
		}
	}
	var f [16]float32
	for i := range f {
		f[i] = float32(i + 2)
	}
	v := archsimd.LoadFloat32x16Array(&f)
	var p, q [16]float32
	v.MulAddEvenSubOdd(v, v).StoreArray(&p)
	v.MulAddOddSubEven(v, v).StoreArray(&q)
	for i, n := range f {
		add := n
		if i%2 == 0 {
			add = -add
		}
		if p[i] != n*n+add || q[i] != n*n-add {
			t.Fatalf("alternating FMA lane %d = %v/%v, want %v/%v", i, p[i], q[i], n*n+add, n*n-add)
		}
	}
}
