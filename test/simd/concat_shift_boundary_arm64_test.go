//go:build goexperiment.simd && arm64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDConcatShiftInvalidCount(t *testing.T) {
	var x archsimd.Uint8x16
	for _, shift := range []uint64{16, 255, 256, 1 << 40} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("shift %d did not panic", shift)
				}
			}()
			_ = x.ConcatShiftBytesRight(x, shift)
		}()
	}
}
