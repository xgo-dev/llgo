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

//go:noinline
func avx2CallsVBMI2(dst, src *[16]int32) bool {
	if archsimd.X86.AVX2() {
		return vbmi2WideCall(dst, src)
	}
	return false
}

//go:noinline
func vbmi2CallsWideKernel(p *[16]int32) {
	if archsimd.X86.AVX512VBMI2() {
		wideKernel(p)
	}
}

//go:noinline
func wideKernel(p *[16]int32) {
	x := archsimd.LoadInt32x16Array(p)
	if archsimd.X86.AVX512() {
		x = vectorcall.Add512(x, x)
	}
	vectorcall.Add512(x, x).StoreArray(p)
}

func TestSIMD512VBMI2WideCall(t *testing.T) {
	avx2, avx512, vbmi2 := archsimd.X86.AVX2(), archsimd.X86.AVX512(), archsimd.X86.AVX512VBMI2()
	t.Logf("AVX2=%t AVX512=%t VBMI2=%t", avx2, avx512, vbmi2)
	var src, doubled [16]int32
	for i := range src {
		src[i] = int32(7*i + 1)
		doubled[i] = 2 * src[i]
	}
	for _, tc := range []struct {
		name string
		call func(*[16]int32, *[16]int32) bool
		take bool
	}{
		{"direct", vbmi2WideCall, vbmi2},
		{"via AVX2", avx2CallsVBMI2, avx2 && vbmi2},
	} {
		var dst, want [16]int32
		if tc.take {
			want = doubled
		}
		bundle := tc.call(&dst, &src)
		if bundle != (tc.take && avx512) || dst != want {
			t.Fatalf("%s: got bundle=%t lanes=%v, want bundle=%t lanes=%v", tc.name, bundle, dst, tc.take && avx512, want)
		}
	}
	t.Run("helper with independent query", func(t *testing.T) {
		got, want := src, src
		vbmi2CallsWideKernel(&got)
		if vbmi2 {
			want = doubled
			if avx512 {
				for i := range want {
					want[i] *= 2
				}
			}
		}
		if got != want {
			t.Fatalf("lanes = %v, want %v", got, want)
		}
	})
}

//go:noinline
func doubleFour(p *[4]int32) {
	x := archsimd.LoadInt32x4Array(p)
	x.Add(x).StoreArray(p)
}

//go:noinline
func independentWideCall(narrow *[4]int32, wide *[16]int32) {
	if archsimd.X86.AVX() {
		doubleFour(narrow)
	}
	if archsimd.X86.FMA() {
		doubleFour(narrow)
	}
	if archsimd.X86.SHA() {
		doubleFour(narrow)
	}
	if archsimd.X86.AVXVNNI() {
		doubleFour(narrow)
	}
	if archsimd.X86.AVX512VBMI2() {
		x := archsimd.LoadInt32x16Array(wide)
		vectorcall.Add512(x, x).StoreArray(wide)
	}
}

func TestSIMD512IndependentGuardsWideCall(t *testing.T) {
	if !archsimd.X86.AVX() {
		t.Skip("128-bit archsimd operations require AVX")
	}
	fma, sha, vnni := archsimd.X86.FMA(), archsimd.X86.SHA(), archsimd.X86.AVXVNNI()
	vbmi2 := archsimd.X86.AVX512VBMI2()
	t.Logf("FMA=%t SHA=%t AVXVNNI=%t VBMI2=%t", fma, sha, vnni, vbmi2)
	narrow := [4]int32{1, -2, 3, -4}
	want := narrow
	for _, enabled := range []bool{true, fma, sha, vnni} {
		if enabled {
			for i := range want {
				want[i] *= 2
			}
		}
	}
	var wide [16]int32
	for i := range wide {
		wide[i] = int32(i + 1)
	}
	independentWideCall(&narrow, &wide)
	if narrow != want {
		t.Fatalf("narrow lanes = %v, want %v", narrow, want)
	}
	for i, got := range wide {
		want := int32(i + 1)
		if vbmi2 {
			want *= 2
		}
		if got != want {
			t.Fatalf("wide lane %d = %d, want %d", i, got, want)
		}
	}
}

//go:noinline
func eitherGuard512(dst *[16]int32, enabled bool, makeVector func() archsimd.Int32x16) {
	if archsimd.X86.AVX512() || archsimd.X86.AVX512VBMI2() {
		if enabled {
			makeVector().StoreArray(dst)
		}
	}
}

func TestSIMD512EitherGuardWideCall(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		var dst, want [16]int32
		var calls, wantCalls int
		eitherGuard512(&dst, enabled, func() archsimd.Int32x16 {
			calls++
			return archsimd.BroadcastInt32x16(13)
		})
		if enabled && (archsimd.X86.AVX512() || archsimd.X86.AVX512VBMI2()) {
			wantCalls = 1
			for i := range want {
				want[i] = 13
			}
		}
		if dst != want || calls != wantCalls {
			t.Fatalf("enabled=%t: got %v and %d calls, want %v and %d", enabled, dst, calls, want, wantCalls)
		}
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
