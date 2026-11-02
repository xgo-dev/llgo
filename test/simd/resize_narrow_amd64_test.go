//go:build goexperiment.simd && llgo && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

// AMD64 unary narrowing requires AVX512 in the official compiler.
// LLGo checks its baseline legalization against scalar references instead.
func TestSIMDNarrowAmd64(t *testing.T) {
	t.Run("Int32x4.SaturateToInt8", func(t *testing.T) {
		testSIMDIntegerResize(t, "saturate", func(in []int32, out []int8) { archsimd.LoadInt32x4(in).SaturateToInt8().Store(out) })
	})
	t.Run("Int64x2.SaturateToInt16", func(t *testing.T) {
		testSIMDIntegerResize(t, "saturate", func(in []int64, out []int16) { archsimd.LoadInt64x2(in).SaturateToInt16().Store(out) })
	})
	t.Run("Int64x2.SaturateToInt8", func(t *testing.T) {
		testSIMDIntegerResize(t, "saturate", func(in []int64, out []int8) { archsimd.LoadInt64x2(in).SaturateToInt8().Store(out) })
	})
	t.Run("Uint32x4.SaturateToUint8", func(t *testing.T) {
		testSIMDIntegerResize(t, "saturate", func(in []uint32, out []uint8) { archsimd.LoadUint32x4(in).SaturateToUint8().Store(out) })
	})
	t.Run("Uint64x2.SaturateToUint16", func(t *testing.T) {
		testSIMDIntegerResize(t, "saturate", func(in []uint64, out []uint16) { archsimd.LoadUint64x2(in).SaturateToUint16().Store(out) })
	})
	t.Run("Uint64x2.SaturateToUint8", func(t *testing.T) {
		testSIMDIntegerResize(t, "saturate", func(in []uint64, out []uint8) { archsimd.LoadUint64x2(in).SaturateToUint8().Store(out) })
	})
}
