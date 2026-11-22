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

func TestSIMDGroupedConcatShiftWindow(t *testing.T) {
	if !archsimd.X86.AVX2() {
		t.Skip("requires AVX2 for the gc control")
	}
	var hi, lo, out [32]byte
	for i := range hi {
		hi[i] = byte(100 + i)
		lo[i] = byte(1 + i)
	}
	x, y := archsimd.LoadUint8x32Array(&hi), archsimd.LoadUint8x32Array(&lo)
	for _, count := range []uint64{0, 1, 15, 16, 17, 31, 32, 255, 256, 257} {
		x.ConcatShiftBytesRightGrouped(y, count).StoreArray(&out)
		for i, v := range out {
			base := i / 16 * 16
			j := i%16 + int(uint8(count))
			var want byte
			switch {
			case j < 16:
				want = lo[base+j]
			case j < 32:
				want = hi[base+j-16]
			}
			if v != want {
				t.Fatalf("shift %d lane %d = %d, want %d", count, i, v, want)
			}
		}
	}
}
