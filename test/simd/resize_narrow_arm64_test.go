//go:build goexperiment.simd && arm64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDNarrowArm64(t *testing.T) {
	t.Run("Int16x8.SaturateToUint8", func(t *testing.T) {
		testSIMDIntegerResize(t, "saturate", func(in []int16, out []uint8) { archsimd.LoadInt16x8(in).SaturateToUint8().Store(out) })
	})
	t.Run("Int32x4.SaturateToUint16", func(t *testing.T) {
		testSIMDIntegerResize(t, "saturate", func(in []int32, out []uint16) { archsimd.LoadInt32x4(in).SaturateToUint16().Store(out) })
	})
	t.Run("Int64x2.SaturateToUint32", func(t *testing.T) {
		testSIMDIntegerResize(t, "saturate", func(in []int64, out []uint32) { archsimd.LoadInt64x2(in).SaturateToUint32().Store(out) })
	})
}
