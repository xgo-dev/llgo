//go:build goexperiment.simd && amd64

package simd_test

import (
	"runtime"
	"simd/archsimd"
	"testing"
)

//go:noinline
func add512(x, y archsimd.Int32x16) archsimd.Int32x16 { return x.Add(y) }

//go:noinline
func invoke512(f func(archsimd.Int32x16, archsimd.Int32x16) archsimd.Int32x16, x, y archsimd.Int32x16) archsimd.Int32x16 {
	return f(x, y)
}

func TestSIMD512Core(t *testing.T) {
	if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
		t.Skip("requires AVX512 in gc")
	}
	var a, b, out [16]int32
	for i := range a {
		a[i] = int32(i*7 + 1)
		b[i] = int32(19 - i)
	}
	x, y := archsimd.LoadInt32x16Array(&a), archsimd.LoadInt32x16Array(&b)
	invoke512(add512, x, y).InterleaveHiGrouped(y).StoreArray(&out)
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
