//go:build goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

//go:noinline
func dynamicRound(x archsimd.Float32x4, precision uint8) (floor, ceil, trunc [4]float32) {
	x.FloorScaled(precision).StoreArray(&floor)
	x.CeilScaled(precision).StoreArray(&ceil)
	x.TruncScaled(precision).StoreArray(&trunc)
	return
}

func TestSIMDX86FixedAndDynamicRound(t *testing.T) {
	if !archsimd.X86.AVX512() {
		t.Skip("requires AVX512")
	}
	in := [4]float32{3.375, -3.375, 0.625, -0.625}
	x := archsimd.LoadFloat32x4Array(&in)
	var floor, ceil, trunc [4]float32
	x.FloorScaled(18).StoreArray(&floor)
	x.CeilScaled(2).StoreArray(&ceil)
	x.TruncScaled(242).StoreArray(&trunc)
	if floor != [4]float32{3.25, -3.5, .5, -.75} || ceil != [4]float32{3.5, -3.25, .75, -.5} || trunc != [4]float32{3.25, -3.25, .5, -.5} {
		t.Fatalf("floor %v, ceil %v, trunc %v", floor, ceil, trunc)
	}
	for _, precision := range []uint8{2, 18, 242} {
		f, c, tr := dynamicRound(x, precision)
		if f != floor || c != ceil || tr != trunc {
			t.Fatalf("precision %d: floor %v, ceil %v, trunc %v", precision, f, c, tr)
		}
	}
	f, c, tr := dynamicRound(x, 0)
	if f != [4]float32{3, -4, 0, -1} || c != [4]float32{4, -3, 1, 0} || tr != [4]float32{3, -3, 0, 0} {
		t.Fatalf("precision zero: floor %v, ceil %v, trunc %v", f, c, tr)
	}
}

func TestSIMDX86GFNIAffine(t *testing.T) {
	if !archsimd.X86.AVX512GFNI() {
		t.Skip("requires AVX512GFNI")
	}
	var in [64]uint8
	for i := range in {
		in[i] = uint8(i*13 + 7)
	}
	identity := archsimd.BroadcastUint64x8(0x0102040810204080)
	var out [64]uint8
	for _, offset := range []byte{0, 0x5a, 0xff} {
		archsimd.LoadUint8x64Array(&in).GaloisFieldAffineTransform(identity, offset).StoreArray(&out)
		for i, x := range in {
			if out[i] != x^offset {
				t.Fatalf("lane %d: %#x, want %#x", i, out[i], x^offset)
			}
		}
	}
}

//go:noinline
func dynamicSHA1(x, words archsimd.Uint32x4, selector uint8) (out [4]uint32) {
	x.SHA1FourRounds(selector, words).StoreArray(&out)
	return
}

func TestSIMDX86FixedAndDynamicSHA1(t *testing.T) {
	if !archsimd.X86.SHA() {
		t.Skip("requires SHA")
	}
	x := archsimd.LoadUint32x4Array(&[4]uint32{0x10325476, 0x98badcfe, 0xefcdab89, 0x67452301})
	words := archsimd.LoadUint32x4Array(&[4]uint32{5, 11, 17, 23})
	var fixed [4][4]uint32
	x.SHA1FourRounds(0, words).StoreArray(&fixed[0])
	x.SHA1FourRounds(1, words).StoreArray(&fixed[1])
	x.SHA1FourRounds(2, words).StoreArray(&fixed[2])
	x.SHA1FourRounds(7, words).StoreArray(&fixed[3])
	for i := 0; i < 256; i++ {
		if got, want := dynamicSHA1(x, words, uint8(i)), fixed[i&3]; got != want {
			t.Fatalf("SHA1 selector %d: got %x, want %x", i, got, want)
		}
	}
}
