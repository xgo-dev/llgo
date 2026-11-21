//go:build goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMD512RoundScale(t *testing.T) {
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
func TestSIMD512AESRound(t *testing.T) {
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
