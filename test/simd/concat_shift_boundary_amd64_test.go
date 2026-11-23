//go:build goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDConcatShiftImmediateByte(t *testing.T) {
	requireSIMD128(t)
	var x, y [16]uint8
	for i := range x {
		y[i] = uint8(i + 1)
		x[i] = uint8(i + 17)
	}
	high, low := archsimd.LoadUint8x16Array(&x), archsimd.LoadUint8x16Array(&y)
	for _, count := range []uint64{0, 15, 16, 31, 32, 255, 256, 257, 1<<40 + 31} {
		var got [16]uint8
		high.ConcatShiftBytesRight(low, count).StoreArray(&got)
		for i, v := range got {
			offset := int(uint8(count)) + i
			var want uint8
			if offset < 32 {
				want = uint8(offset + 1)
			}
			if v != want {
				t.Fatalf("shift %d lane %d: got %d want %d", count, i, v, want)
			}
		}
	}
}
