//go:build goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDExtendAmd64(t *testing.T) {
	t.Run("Int16x8.ExtendLo2ToInt64", func(t *testing.T) {
		testSIMDIntegerResize(t, "lo", func(in []int16, out []int64) { archsimd.LoadInt16x8(in).ExtendLo2ToInt64().Store(out) })
	})
	t.Run("Int8x16.ExtendLo2ToInt64", func(t *testing.T) {
		testSIMDIntegerResize(t, "lo", func(in []int8, out []int64) { archsimd.LoadInt8x16(in).ExtendLo2ToInt64().Store(out) })
	})
	t.Run("Int8x16.ExtendLo4ToInt32", func(t *testing.T) {
		testSIMDIntegerResize(t, "lo", func(in []int8, out []int32) { archsimd.LoadInt8x16(in).ExtendLo4ToInt32().Store(out) })
	})
	t.Run("Uint16x8.ExtendLo2ToUint64", func(t *testing.T) {
		testSIMDIntegerResize(t, "lo", func(in []uint16, out []uint64) { archsimd.LoadUint16x8(in).ExtendLo2ToUint64().Store(out) })
	})
	t.Run("Uint8x16.ExtendLo2ToUint64", func(t *testing.T) {
		testSIMDIntegerResize(t, "lo", func(in []uint8, out []uint64) { archsimd.LoadUint8x16(in).ExtendLo2ToUint64().Store(out) })
	})
	t.Run("Uint8x16.ExtendLo4ToUint32", func(t *testing.T) {
		testSIMDIntegerResize(t, "lo", func(in []uint8, out []uint32) { archsimd.LoadUint8x16(in).ExtendLo4ToUint32().Store(out) })
	})
}
