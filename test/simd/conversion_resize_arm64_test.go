//go:build goexperiment.simd && arm64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDConversionResize(t *testing.T) {
	t.Run("Float32x4/ConvertLo2ToFloat64", func(t *testing.T) {
		testSIMDConversionShape[float32, float64](t, "ConvertLo2ToFloat64", 4, 2, func(x []float32, out []float64) {
			a := archsimd.LoadFloat32x4Array((*[4]float32)(x))
			a.ConvertLo2ToFloat64().StoreArray((*[2]float64)(out))
		})
	})
	t.Run("Float64x2/ConvertToFloat32", func(t *testing.T) {
		testSIMDConversionShape[float64, float32](t, "ConvertToFloat32", 2, 4, func(x []float64, out []float32) {
			a := archsimd.LoadFloat64x2Array((*[2]float64)(x))
			a.ConvertToFloat32().StoreArray((*[4]float32)(out))
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
	t.Run("Int16x8/SaturateToInt8", func(t *testing.T) {
		testSIMDIntegerResizeShape[int16, int8](t, "saturate", 8, 16, func(x []int16, out []int8) {
			a := archsimd.LoadInt16x8Array((*[8]int16)(x))
			a.SaturateToInt8().StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int32x4/SaturateToInt16", func(t *testing.T) {
		testSIMDIntegerResizeShape[int32, int16](t, "saturate", 4, 8, func(x []int32, out []int16) {
			a := archsimd.LoadInt32x4Array((*[4]int32)(x))
			a.SaturateToInt16().StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int64x2/SaturateToInt32", func(t *testing.T) {
		testSIMDIntegerResizeShape[int64, int32](t, "saturate", 2, 4, func(x []int64, out []int32) {
			a := archsimd.LoadInt64x2Array((*[2]int64)(x))
			a.SaturateToInt32().StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Int16x8/SaturateToUint8", func(t *testing.T) {
		testSIMDIntegerResizeShape[int16, uint8](t, "saturate", 8, 16, func(x []int16, out []uint8) {
			a := archsimd.LoadInt16x8Array((*[8]int16)(x))
			a.SaturateToUint8().StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint16x8/SaturateToUint8", func(t *testing.T) {
		testSIMDIntegerResizeShape[uint16, uint8](t, "saturate", 8, 16, func(x []uint16, out []uint8) {
			a := archsimd.LoadUint16x8Array((*[8]uint16)(x))
			a.SaturateToUint8().StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Int32x4/SaturateToUint16", func(t *testing.T) {
		testSIMDIntegerResizeShape[int32, uint16](t, "saturate", 4, 8, func(x []int32, out []uint16) {
			a := archsimd.LoadInt32x4Array((*[4]int32)(x))
			a.SaturateToUint16().StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint32x4/SaturateToUint16", func(t *testing.T) {
		testSIMDIntegerResizeShape[uint32, uint16](t, "saturate", 4, 8, func(x []uint32, out []uint16) {
			a := archsimd.LoadUint32x4Array((*[4]uint32)(x))
			a.SaturateToUint16().StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Int64x2/SaturateToUint32", func(t *testing.T) {
		testSIMDIntegerResizeShape[int64, uint32](t, "saturate", 2, 4, func(x []int64, out []uint32) {
			a := archsimd.LoadInt64x2Array((*[2]int64)(x))
			a.SaturateToUint32().StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Uint64x2/SaturateToUint32", func(t *testing.T) {
		testSIMDIntegerResizeShape[uint64, uint32](t, "saturate", 2, 4, func(x []uint64, out []uint32) {
			a := archsimd.LoadUint64x2Array((*[2]uint64)(x))
			a.SaturateToUint32().StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Int16x8/TruncToInt8", func(t *testing.T) {
		testSIMDIntegerResizeShape[int16, int8](t, "truncate", 8, 16, func(x []int16, out []int8) {
			a := archsimd.LoadInt16x8Array((*[8]int16)(x))
			a.TruncToInt8().StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int32x4/TruncToInt16", func(t *testing.T) {
		testSIMDIntegerResizeShape[int32, int16](t, "truncate", 4, 8, func(x []int32, out []int16) {
			a := archsimd.LoadInt32x4Array((*[4]int32)(x))
			a.TruncToInt16().StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int64x2/TruncToInt32", func(t *testing.T) {
		testSIMDIntegerResizeShape[int64, int32](t, "truncate", 2, 4, func(x []int64, out []int32) {
			a := archsimd.LoadInt64x2Array((*[2]int64)(x))
			a.TruncToInt32().StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Uint16x8/TruncToUint8", func(t *testing.T) {
		testSIMDIntegerResizeShape[uint16, uint8](t, "truncate", 8, 16, func(x []uint16, out []uint8) {
			a := archsimd.LoadUint16x8Array((*[8]uint16)(x))
			a.TruncToUint8().StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint32x4/TruncToUint16", func(t *testing.T) {
		testSIMDIntegerResizeShape[uint32, uint16](t, "truncate", 4, 8, func(x []uint32, out []uint16) {
			a := archsimd.LoadUint32x4Array((*[4]uint32)(x))
			a.TruncToUint16().StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint64x2/TruncToUint32", func(t *testing.T) {
		testSIMDIntegerResizeShape[uint64, uint32](t, "truncate", 2, 4, func(x []uint64, out []uint32) {
			a := archsimd.LoadUint64x2Array((*[2]uint64)(x))
			a.TruncToUint32().StoreArray((*[4]uint32)(out))
		})
	})
}
