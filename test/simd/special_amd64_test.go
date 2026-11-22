//go:build goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDRoundScale(t *testing.T) {
	if !archsimd.X86.AVX512() {
		t.Skip("requires AVX512")
	}
	in := [4]float32{3.375, -3.375, 0.625, -0.625}
	x := archsimd.LoadFloat32x4Array(&in)
	for _, precision := range []uint8{2, 18, 242} {
		var round, residue [4]float32
		x.RoundScaled(precision).StoreArray(&round)
		x.RoundScaledResidue(precision).StoreArray(&residue)
		if round != [4]float32{3.5, -3.5, 0.5, -0.5} || residue != [4]float32{-0.125, 0.125, 0.125, -0.125} {
			t.Fatalf("precision %d: round=%v residue=%v", precision, round, residue)
		}
	}
}
func TestSIMDAESRound(t *testing.T) {
	if !archsimd.X86.AVXAES() {
		t.Skip("requires AVX and AES")
	}
	var x archsimd.Uint8x16
	var key archsimd.Uint32x4
	var result [16]uint8
	x.AESEncryptOneRound(key).StoreArray(&result)
	for i, b := range result {
		if b != 0x63 {
			t.Fatalf("AES zero round lane %d = %#x", i, b)
		}
	}
	for i := 0; i < 256; i++ {
		var words [4]uint32
		key.AESRoundKeyGenAssist(uint8(i)).StoreArray(&words)
		want := [4]uint32{0x63636363, 0x63636363 ^ uint32(i), 0x63636363, 0x63636363 ^ uint32(i)}
		if words != want {
			t.Fatalf("Rcon %d = %x, want %x", i, words, want)
		}
	}
}

//go:noinline
func dynamicRound(x archsimd.Float32x4, precision uint8) (floor, ceil, trunc [4]float32) {
	x.FloorScaled(precision).StoreArray(&floor)
	x.CeilScaled(precision).StoreArray(&ceil)
	x.TruncScaled(precision).StoreArray(&trunc)
	return
}

func TestSIMDFixedAndDynamicRound(t *testing.T) {
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

func TestSIMDGFNIAffine(t *testing.T) {
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
