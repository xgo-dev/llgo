//go:build goexperiment.simd && arm64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDSignedLowByteShift(t *testing.T) {
	x := [4]int32{-1, -123, 0x12345678, -2147483648}
	a := archsimd.LoadInt32x4Array(&x)
	for _, counts := range [][4]int32{{0, 31, -32, 257}, {-1, 255, 128, 127}, {-129, 256, -256, -255}} {
		c := archsimd.LoadInt32x4Array(&counts)
		var signed [4]int32
		var unsigned [4]uint32
		a.Shift(c).StoreArray(&signed)
		a.ToBits().Shift(c).StoreArray(&unsigned)
		for i, raw := range counts {
			n := int(int8(raw))
			var want int32
			var wantU uint32
			if n < 0 {
				want = x[i] >> uint(-n)
				wantU = uint32(x[i]) >> uint(-n)
			} else {
				want = x[i] << uint(n)
				wantU = uint32(x[i]) << uint(n)
			}
			if signed[i] != want || unsigned[i] != wantU {
				t.Fatalf("count %d lane %d: %x %x want %x %x", raw, i, signed[i], unsigned[i], want, wantU)
			}
		}
	}
}
