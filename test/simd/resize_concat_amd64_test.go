//go:build goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDNarrowConcat(t *testing.T) {
	t.Run("Int32x4.SaturateToInt16Concat", func(t *testing.T) {
		testSIMDIntegerResize(t, "concat", func(in []int32, out []int16) {
			archsimd.LoadInt32x4(in).SaturateToInt16Concat(archsimd.LoadInt32x4(in[4:])).Store(out)
		})
	})
	t.Run("Int32x4.SaturateToUint16Concat", func(t *testing.T) {
		testSIMDIntegerResize(t, "concat", func(in []int32, out []uint16) {
			archsimd.LoadInt32x4(in).SaturateToUint16Concat(archsimd.LoadInt32x4(in[4:])).Store(out)
		})
	})
}
