//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDExtend(t *testing.T) {
	t.Run("Int16x8.ExtendLo4ToInt32", func(t *testing.T) {
		testSIMDIntegerResize(t, "lo", func(in []int16, out []int32) { archsimd.LoadInt16x8(in).ExtendLo4ToInt32().Store(out) })
	})
	t.Run("Int32x4.ExtendLo2ToInt64", func(t *testing.T) {
		testSIMDIntegerResize(t, "lo", func(in []int32, out []int64) { archsimd.LoadInt32x4(in).ExtendLo2ToInt64().Store(out) })
	})
	t.Run("Int8x16.ExtendLo8ToInt16", func(t *testing.T) {
		testSIMDIntegerResize(t, "lo", func(in []int8, out []int16) { archsimd.LoadInt8x16(in).ExtendLo8ToInt16().Store(out) })
	})
	t.Run("Uint16x8.ExtendLo4ToUint32", func(t *testing.T) {
		testSIMDIntegerResize(t, "lo", func(in []uint16, out []uint32) { archsimd.LoadUint16x8(in).ExtendLo4ToUint32().Store(out) })
	})
	t.Run("Uint32x4.ExtendLo2ToUint64", func(t *testing.T) {
		testSIMDIntegerResize(t, "lo", func(in []uint32, out []uint64) { archsimd.LoadUint32x4(in).ExtendLo2ToUint64().Store(out) })
	})
	t.Run("Uint8x16.ExtendLo8ToUint16", func(t *testing.T) {
		testSIMDIntegerResize(t, "lo", func(in []uint8, out []uint16) { archsimd.LoadUint8x16(in).ExtendLo8ToUint16().Store(out) })
	})
}
