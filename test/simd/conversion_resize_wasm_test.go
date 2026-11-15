//go:build goexperiment.simd && wasm

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDConversionResize(t *testing.T) {
	t.Run("Int32x4/ConvertLo2ToFloat64", func(t *testing.T) {
		testSIMDConversionShape[int32, float64](t, "ConvertLo2ToFloat64", 4, 2, func(x []int32, out []float64) {
			a := archsimd.LoadInt32x4Array((*[4]int32)(x))
			a.ConvertLo2ToFloat64().StoreArray((*[2]float64)(out))
		})
	})
	t.Run("Uint32x4/ConvertLo2ToFloat64", func(t *testing.T) {
		testSIMDConversionShape[uint32, float64](t, "ConvertLo2ToFloat64", 4, 2, func(x []uint32, out []float64) {
			a := archsimd.LoadUint32x4Array((*[4]uint32)(x))
			a.ConvertLo2ToFloat64().StoreArray((*[2]float64)(out))
		})
	})
	t.Run("Int32x4/ExtendHi2ToInt64", func(t *testing.T) {
		testSIMDIntegerResizeShape[int32, int64](t, "hi", 4, 2, func(x []int32, out []int64) {
			a := archsimd.LoadInt32x4Array((*[4]int32)(x))
			a.ExtendHi2ToInt64().StoreArray((*[2]int64)(out))
		})
	})
	t.Run("Uint32x4/ExtendHi2ToUint64", func(t *testing.T) {
		testSIMDIntegerResizeShape[uint32, uint64](t, "hi", 4, 2, func(x []uint32, out []uint64) {
			a := archsimd.LoadUint32x4Array((*[4]uint32)(x))
			a.ExtendHi2ToUint64().StoreArray((*[2]uint64)(out))
		})
	})
	t.Run("Int16x8/ExtendHi4ToInt32", func(t *testing.T) {
		testSIMDIntegerResizeShape[int16, int32](t, "hi", 8, 4, func(x []int16, out []int32) {
			a := archsimd.LoadInt16x8Array((*[8]int16)(x))
			a.ExtendHi4ToInt32().StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Uint16x8/ExtendHi4ToUint32", func(t *testing.T) {
		testSIMDIntegerResizeShape[uint16, uint32](t, "hi", 8, 4, func(x []uint16, out []uint32) {
			a := archsimd.LoadUint16x8Array((*[8]uint16)(x))
			a.ExtendHi4ToUint32().StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Int8x16/ExtendHi8ToInt16", func(t *testing.T) {
		testSIMDIntegerResizeShape[int8, int16](t, "hi", 16, 8, func(x []int8, out []int16) {
			a := archsimd.LoadInt8x16Array((*[16]int8)(x))
			a.ExtendHi8ToInt16().StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Uint8x16/ExtendHi8ToUint16", func(t *testing.T) {
		testSIMDIntegerResizeShape[uint8, uint16](t, "hi", 16, 8, func(x []uint8, out []uint16) {
			a := archsimd.LoadUint8x16Array((*[16]uint8)(x))
			a.ExtendHi8ToUint16().StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Int32x4/ExtendLo2ToInt64", func(t *testing.T) {
		testSIMDIntegerResizeShape[int32, int64](t, "lo", 4, 2, func(x []int32, out []int64) {
			a := archsimd.LoadInt32x4Array((*[4]int32)(x))
			a.ExtendLo2ToInt64().StoreArray((*[2]int64)(out))
		})
	})
	t.Run("Uint32x4/ExtendLo2ToUint64", func(t *testing.T) {
		testSIMDIntegerResizeShape[uint32, uint64](t, "lo", 4, 2, func(x []uint32, out []uint64) {
			a := archsimd.LoadUint32x4Array((*[4]uint32)(x))
			a.ExtendLo2ToUint64().StoreArray((*[2]uint64)(out))
		})
	})
	t.Run("Int16x8/ExtendLo4ToInt32", func(t *testing.T) {
		testSIMDIntegerResizeShape[int16, int32](t, "lo", 8, 4, func(x []int16, out []int32) {
			a := archsimd.LoadInt16x8Array((*[8]int16)(x))
			a.ExtendLo4ToInt32().StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Uint16x8/ExtendLo4ToUint32", func(t *testing.T) {
		testSIMDIntegerResizeShape[uint16, uint32](t, "lo", 8, 4, func(x []uint16, out []uint32) {
			a := archsimd.LoadUint16x8Array((*[8]uint16)(x))
			a.ExtendLo4ToUint32().StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Int8x16/ExtendLo8ToInt16", func(t *testing.T) {
		testSIMDIntegerResizeShape[int8, int16](t, "lo", 16, 8, func(x []int8, out []int16) {
			a := archsimd.LoadInt8x16Array((*[16]int8)(x))
			a.ExtendLo8ToInt16().StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Uint8x16/ExtendLo8ToUint16", func(t *testing.T) {
		testSIMDIntegerResizeShape[uint8, uint16](t, "lo", 16, 8, func(x []uint8, out []uint16) {
			a := archsimd.LoadUint8x16Array((*[16]uint8)(x))
			a.ExtendLo8ToUint16().StoreArray((*[8]uint16)(out))
		})
	})
}
