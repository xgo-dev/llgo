//go:build goexperiment.simd && arm64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDOrNot(t *testing.T) {
	x, y := [4]int32{0, -1, 0x12345678, -17}, [4]int32{-1, 0, -17, 0x12345678}
	var got [4]int32
	archsimd.LoadInt32x4Array(&x).OrNot(archsimd.LoadInt32x4Array(&y)).StoreArray(&got)
	for i := range x {
		if got[i] != x[i]|^y[i] {
			t.Fatalf("lane %d = %x", i, got[i])
		}
	}
}
