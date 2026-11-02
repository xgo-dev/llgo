//go:build goexperiment.simd && (arm64 || (llgo && amd64))

package simd_test

import (
	"simd/archsimd"
	"testing"
)

// AMD64 unary narrowing requires AVX512 in the official compiler.
// LLGo checks its baseline legalization against scalar references instead.
func TestSIMDNarrowNative(t *testing.T) {
	t.Run("Int16x8.SaturateToInt8", func(t *testing.T) {
		testSIMDIntegerResize(t, "saturate", func(in []int16, out []int8) { archsimd.LoadInt16x8(in).SaturateToInt8().Store(out) })
	})
	t.Run("Int32x4.SaturateToInt16", func(t *testing.T) {
		testSIMDIntegerResize(t, "saturate", func(in []int32, out []int16) { archsimd.LoadInt32x4(in).SaturateToInt16().Store(out) })
	})
	t.Run("Int64x2.SaturateToInt32", func(t *testing.T) {
		testSIMDIntegerResize(t, "saturate", func(in []int64, out []int32) { archsimd.LoadInt64x2(in).SaturateToInt32().Store(out) })
	})
	t.Run("Uint16x8.SaturateToUint8", func(t *testing.T) {
		testSIMDIntegerResize(t, "saturate", func(in []uint16, out []uint8) { archsimd.LoadUint16x8(in).SaturateToUint8().Store(out) })
	})
	t.Run("Uint32x4.SaturateToUint16", func(t *testing.T) {
		testSIMDIntegerResize(t, "saturate", func(in []uint32, out []uint16) { archsimd.LoadUint32x4(in).SaturateToUint16().Store(out) })
	})
	t.Run("Uint64x2.SaturateToUint32", func(t *testing.T) {
		testSIMDIntegerResize(t, "saturate", func(in []uint64, out []uint32) { archsimd.LoadUint64x2(in).SaturateToUint32().Store(out) })
	})
}
