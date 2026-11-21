//go:build goexperiment.simd && amd64

package simd_test

import (
	"runtime"
	"simd/archsimd"
	"testing"
)

func TestSIMD512Rearrange(t *testing.T) {
	if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
		t.Skip("requires AVX512 in gc")
	}
	var a, b, indices, out [16]uint32
	for i := range a {
		a[i] = uint32(i + 1)
		b[i] = uint32(400 + i)
		indices[i] = 0xffffffe0 + uint32(i*11%32)
	}
	x, y := archsimd.LoadUint32x16Array(&a), archsimd.LoadUint32x16Array(&b)
	x.ConcatPermute(y, archsimd.LoadUint32x16Array(&indices)).StoreArray(&out)
	for i, got := range out {
		j := indices[i] & 31
		want := a[j&15]
		if j >= 16 {
			want = b[j&15]
		}
		if got != want {
			t.Fatalf("permute lane %d = %d, want %d", i, got, want)
		}
	}
	for _, bits := range []uint16{0, 1, 0x8126, 0xaaaa, 0xffff} {
		m := archsimd.Mask32x16FromBits(bits)
		var packed, expanded [16]uint32
		rank := 0
		for i := range a {
			if bits&(1<<i) != 0 {
				packed[rank] = a[i]
				expanded[i] = a[rank]
				rank++
			}
		}
		x.Compress(m).StoreArray(&out)
		if out != packed {
			t.Fatalf("compress mask %#x = %v, want %v", bits, out, packed)
		}
		x.Expand(m).StoreArray(&out)
		if out != expanded {
			t.Fatalf("expand mask %#x = %v, want %v", bits, out, expanded)
		}
	}
}
