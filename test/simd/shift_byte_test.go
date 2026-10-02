//go:build goexperiment.simd && (arm64 || wasm)

package simd_test

import (
	"math"
	"simd/archsimd"
	"testing"
)

func TestSIMDByteShift(t *testing.T) {
	x8 := [16]int8{math.MinInt8, math.MaxInt8, -1, 0, 1, -2, 2, 3, -3, 4, -4, 5, -5, 6, -6, 7}
	for _, n := range []uint64{0, 1, 7, 8, 9, 127, 128, 255, 256, math.MaxUint64} {
		var left8, right8 [16]int8
		v8 := archsimd.LoadInt8x16Array(&x8)
		v8.ShiftAllLeft(n).StoreArray(&left8)
		v8.ShiftAllRight(n).StoreArray(&right8)
		for i, x := range x8 {
			if left8[i] != x<<n || right8[i] != x>>n {
				t.Fatalf("int8 shift %d lane %d: %d %d", n, i, left8[i], right8[i])
			}
		}

	}
}
