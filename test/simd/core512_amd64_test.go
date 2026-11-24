//go:build goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"

	"github.com/xgo-dev/llgo/test/simd/internal/vectorcall"
)

//go:noinline
func invoke512(f func(archsimd.Int32x16, archsimd.Int32x16) archsimd.Int32x16, x, y archsimd.Int32x16) archsimd.Int32x16 {
	return f(x, y)
}

func TestSIMD512Core(t *testing.T) {
	if !archsimd.X86.AVX512() {
		t.Skip("requires AVX512")
	}
	var a, b, out [16]int32
	for i := range a {
		a[i] = int32(i*7 + 1)
		b[i] = int32(19 - i)
	}
	x, y := archsimd.LoadInt32x16Array(&a), archsimd.LoadInt32x16Array(&b)
	invoke512(vectorcall.Add512, x, y).InterleaveHiGrouped(y).StoreArray(&out)
	for i, got := range out {
		j := i/4*4 + 2 + (i%4)/2
		want := b[j]
		if i%2 == 0 {
			want += a[j]
		}
		if got != want {
			t.Fatalf("lane %d = %d, want %d", i, got, want)
		}
	}
	const selected uint16 = 0xa153
	x.IfElse(archsimd.Mask32x16FromBits(selected), y).StoreArray(&out)
	for i, got := range out {
		want := b[i]
		if selected&(1<<i) != 0 {
			want = a[i]
		}
		if got != want {
			t.Fatalf("blend lane %d = %d, want %d", i, got, want)
		}
	}
	const bits uint64 = 0x8102040810204081
	if got := archsimd.Mask8x64FromBits(bits).ToBits(); got != bits {
		t.Fatalf("mask = %#x, want %#x", got, bits)
	}
}

//go:noinline
func vbmi2WideCall(dst, src *[16]int32) bool {
	if archsimd.X86.AVX512VBMI2() {
		x := archsimd.LoadInt32x16Array(src)
		vectorcall.Add512(x, x).StoreArray(dst)
		return archsimd.X86.AVX512()
	}
	return false
}

func TestSIMD512VBMI2WideCall(t *testing.T) {
	var src, dst, want [16]int32
	for i := range src {
		src[i] = int32(7*i + 1)
	}
	if archsimd.X86.AVX512VBMI2() {
		for i, x := range src {
			want[i] = 2 * x
		}
	}
	bundle := vbmi2WideCall(&dst, &src)
	if bundle != (archsimd.X86.AVX512VBMI2() && archsimd.X86.AVX512()) || dst != want {
		t.Fatalf("AVX512=%t VBMI2=%t: got bundle=%t lanes=%v, want lanes=%v",
			archsimd.X86.AVX512(), archsimd.X86.AVX512VBMI2(), bundle, dst, want)
	}
}

// Nine live vectors also exercise the stack argument path in the register ABI.
//
//go:noinline
func sumNine(a, b, c, d, e, f, g, h, i archsimd.Int32x16) archsimd.Int32x16 {
	return a.Add(b).Add(c).Add(d).Add(e).Add(f).Add(g).Add(h).Add(i)
}

//go:noinline
func pairWide(x archsimd.Int32x16) (archsimd.Int32x16, archsimd.Int32x16) {
	return x, x.Add(x)
}

func TestSIMD512StackArgumentsAndResults(t *testing.T) {
	if !archsimd.X86.AVX512() {
		t.Skip("requires AVX512")
		return
	}
	var in, out [16]int32
	for i := range in {
		in[i] = 0x12340000 + int32(i)*97
	}
	x := archsimd.LoadInt32x16Array(&in)
	a, b := pairWide(x)
	one := archsimd.BroadcastInt32x16(1)
	y := sumNine(a, b, one, one, one, one, one, one, one)
	vectorcall.Add512(y, y).StoreArray(&out)
	for i, v := range out {
		if want := 6*in[i] + 14; v != want {
			t.Errorf("lane %d = %d, want %d", i, v, want)
		}
	}
}
