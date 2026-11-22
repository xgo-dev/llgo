//go:build goexperiment.simd && amd64

package simd_test

import (
	"github.com/xgo-dev/llgo/test/simd/internal/regkernel"
	"simd/archsimd"
	"testing"
)

// Nine live vectors also exercise the stack argument path in the register ABI.
//
//go:noinline
func sumNine(a, b, c, d, e, f, g, h, i archsimd.Uint32x16) archsimd.Uint32x16 {
	return a.Add(b).Add(c).Add(d).Add(e).Add(f).Add(g).Add(h).Add(i)
}

//go:noinline
func pairWide(x archsimd.Uint32x16) (archsimd.Uint32x16, archsimd.Uint32x16) {
	return x, x.Add(x)
}

func TestSIMDWideRegisterCalls(t *testing.T) {
	if !archsimd.X86.AVX512() {
		t.Skip("requires AVX512")
		return
	}
	var in, out [16]uint32
	for i := range in {
		in[i] = 0x12340000 + uint32(i)*97
	}
	x := archsimd.LoadUint32x16Array(&in)
	a, b := pairWide(x)
	one := archsimd.BroadcastUint32x16(1)
	y := sumNine(a, b, one, one, one, one, one, one, one)
	regkernel.Add512(y, y).StoreArray(&out)
	for i, v := range out {
		if want := 6*in[i] + 14; v != want {
			t.Errorf("lane %d = %d, want %d", i, v, want)
		}
	}
}

func TestSIMDWideRegisterImport256(t *testing.T) {
	if !archsimd.X86.AVX2() {
		t.Skip("requires AVX2")
		return
	}
	var x, y, out [8]uint32
	for i := range x {
		x[i] = uint32(i) + 13
		y[i] = uint32(2*i) + 29
	}
	regkernel.Add256(archsimd.LoadUint32x8Array(&x), archsimd.LoadUint32x8Array(&y)).StoreArray(&out)
	for i, v := range out {
		if v != x[i]+y[i] {
			t.Errorf("lane %d = %d", i, v)
		}
	}
}
