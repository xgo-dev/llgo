//go:build goexperiment.simd && wasm

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDExtendWasm(t *testing.T) {
	t.Run("Int16x8.ExtendHi4ToInt32", func(t *testing.T) {
		testSIMDIntegerResize(t, "hi", func(in []int16, out []int32) { archsimd.LoadInt16x8(in).ExtendHi4ToInt32().Store(out) })
	})
	t.Run("Int32x4.ExtendHi2ToInt64", func(t *testing.T) {
		testSIMDIntegerResize(t, "hi", func(in []int32, out []int64) { archsimd.LoadInt32x4(in).ExtendHi2ToInt64().Store(out) })
	})
	t.Run("Int8x16.ExtendHi8ToInt16", func(t *testing.T) {
		testSIMDIntegerResize(t, "hi", func(in []int8, out []int16) { archsimd.LoadInt8x16(in).ExtendHi8ToInt16().Store(out) })
	})
	t.Run("Uint16x8.ExtendHi4ToUint32", func(t *testing.T) {
		testSIMDIntegerResize(t, "hi", func(in []uint16, out []uint32) { archsimd.LoadUint16x8(in).ExtendHi4ToUint32().Store(out) })
	})
	t.Run("Uint32x4.ExtendHi2ToUint64", func(t *testing.T) {
		testSIMDIntegerResize(t, "hi", func(in []uint32, out []uint64) { archsimd.LoadUint32x4(in).ExtendHi2ToUint64().Store(out) })
	})
	t.Run("Uint8x16.ExtendHi8ToUint16", func(t *testing.T) {
		testSIMDIntegerResize(t, "hi", func(in []uint8, out []uint16) { archsimd.LoadUint8x16(in).ExtendHi8ToUint16().Store(out) })
	})
}
