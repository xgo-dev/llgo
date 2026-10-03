//go:build goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDVariableShift(t *testing.T) {
	x := [4]int32{-1, -123, 0x12345678, -2147483648}
	a := archsimd.LoadInt32x4Array(&x)
	for _, counts := range [][4]uint32{{0, 31, 32, 33}, {255, 256, 1 << 31, 0xffffffff}} {
		c := archsimd.LoadUint32x4Array(&counts)
		var left, right [4]int32
		var unsigned [4]uint32
		a.ShiftLeft(c).StoreArray(&left)
		a.ShiftRight(c).StoreArray(&right)
		a.ToBits().ShiftRight(c).StoreArray(&unsigned)
		for i, n := range counts {
			if left[i] != x[i]<<n || right[i] != x[i]>>n || unsigned[i] != uint32(x[i])>>n {
				t.Fatalf("count %d lane %d", n, i)
			}
		}
	}
}
