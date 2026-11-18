//go:build goexperiment.simd && amd64

package simd_test

import (
	"runtime"
	"simd/archsimd"
	"testing"
)

func TestSIMDConversionResize(t *testing.T) {
	t.Run("Float32x8/Ceil", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDConversionShape[float32, float32](t, "Ceil", 8, 8, func(x []float32, out []float32) {
			a := archsimd.LoadFloat32x8Array((*[8]float32)(x))
			a.Ceil().StoreArray((*[8]float32)(out))
		})
	})
	t.Run("Float64x4/Ceil", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDConversionShape[float64, float64](t, "Ceil", 4, 4, func(x []float64, out []float64) {
			a := archsimd.LoadFloat64x4Array((*[4]float64)(x))
			a.Ceil().StoreArray((*[4]float64)(out))
		})
	})
	t.Run("Float64x2/ConvertToFloat32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDConversionShape[float64, float32](t, "ConvertToFloat32", 2, 4, func(x []float64, out []float32) {
			a := archsimd.LoadFloat64x2Array((*[2]float64)(x))
			a.ConvertToFloat32().StoreArray((*[4]float32)(out))
		})
	})
	t.Run("Float64x4/ConvertToFloat32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDConversionShape[float64, float32](t, "ConvertToFloat32", 4, 4, func(x []float64, out []float32) {
			a := archsimd.LoadFloat64x4Array((*[4]float64)(x))
			a.ConvertToFloat32().StoreArray((*[4]float32)(out))
		})
	})
	t.Run("Int64x2/ConvertToFloat32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDConversionShape[int64, float32](t, "ConvertToFloat32", 2, 4, func(x []int64, out []float32) {
			a := archsimd.LoadInt64x2Array((*[2]int64)(x))
			a.ConvertToFloat32().StoreArray((*[4]float32)(out))
		})
	})
	t.Run("Int64x4/ConvertToFloat32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDConversionShape[int64, float32](t, "ConvertToFloat32", 4, 4, func(x []int64, out []float32) {
			a := archsimd.LoadInt64x4Array((*[4]int64)(x))
			a.ConvertToFloat32().StoreArray((*[4]float32)(out))
		})
	})
	t.Run("Uint64x2/ConvertToFloat32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDConversionShape[uint64, float32](t, "ConvertToFloat32", 2, 4, func(x []uint64, out []float32) {
			a := archsimd.LoadUint64x2Array((*[2]uint64)(x))
			a.ConvertToFloat32().StoreArray((*[4]float32)(out))
		})
	})
	t.Run("Uint64x4/ConvertToFloat32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDConversionShape[uint64, float32](t, "ConvertToFloat32", 4, 4, func(x []uint64, out []float32) {
			a := archsimd.LoadUint64x4Array((*[4]uint64)(x))
			a.ConvertToFloat32().StoreArray((*[4]float32)(out))
		})
	})
	t.Run("Float32x4/ConvertToFloat64", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDConversionShape[float32, float64](t, "ConvertToFloat64", 4, 4, func(x []float32, out []float64) {
			a := archsimd.LoadFloat32x4Array((*[4]float32)(x))
			a.ConvertToFloat64().StoreArray((*[4]float64)(out))
		})
	})
	t.Run("Int32x4/ConvertToFloat64", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDConversionShape[int32, float64](t, "ConvertToFloat64", 4, 4, func(x []int32, out []float64) {
			a := archsimd.LoadInt32x4Array((*[4]int32)(x))
			a.ConvertToFloat64().StoreArray((*[4]float64)(out))
		})
	})
	t.Run("Uint32x4/ConvertToFloat64", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDConversionShape[uint32, float64](t, "ConvertToFloat64", 4, 4, func(x []uint32, out []float64) {
			a := archsimd.LoadUint32x4Array((*[4]uint32)(x))
			a.ConvertToFloat64().StoreArray((*[4]float64)(out))
		})
	})
	t.Run("Float64x2/ConvertToInt32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDConversionShape[float64, int32](t, "ConvertToInt32", 2, 4, func(x []float64, out []int32) {
			a := archsimd.LoadFloat64x2Array((*[2]float64)(x))
			a.ConvertToInt32().StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Float64x4/ConvertToInt32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDConversionShape[float64, int32](t, "ConvertToInt32", 4, 4, func(x []float64, out []int32) {
			a := archsimd.LoadFloat64x4Array((*[4]float64)(x))
			a.ConvertToInt32().StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Float32x4/ConvertToInt64", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDConversionShape[float32, int64](t, "ConvertToInt64", 4, 4, func(x []float32, out []int64) {
			a := archsimd.LoadFloat32x4Array((*[4]float32)(x))
			a.ConvertToInt64().StoreArray((*[4]int64)(out))
		})
	})
	t.Run("Float64x2/ConvertToUint32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDConversionShape[float64, uint32](t, "ConvertToUint32", 2, 4, func(x []float64, out []uint32) {
			a := archsimd.LoadFloat64x2Array((*[2]float64)(x))
			a.ConvertToUint32().StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Float64x4/ConvertToUint32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDConversionShape[float64, uint32](t, "ConvertToUint32", 4, 4, func(x []float64, out []uint32) {
			a := archsimd.LoadFloat64x4Array((*[4]float64)(x))
			a.ConvertToUint32().StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Float32x4/ConvertToUint64", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDConversionShape[float32, uint64](t, "ConvertToUint64", 4, 4, func(x []float32, out []uint64) {
			a := archsimd.LoadFloat32x4Array((*[4]float32)(x))
			a.ConvertToUint64().StoreArray((*[4]uint64)(out))
		})
	})
	t.Run("Int8x16/ExtendLo2ToInt64", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDIntegerResizeShape[int8, int64](t, "lo", 16, 2, func(x []int8, out []int64) {
			a := archsimd.LoadInt8x16Array((*[16]int8)(x))
			a.ExtendLo2ToInt64().StoreArray((*[2]int64)(out))
		})
	})
	t.Run("Int16x8/ExtendLo2ToInt64", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDIntegerResizeShape[int16, int64](t, "lo", 8, 2, func(x []int16, out []int64) {
			a := archsimd.LoadInt16x8Array((*[8]int16)(x))
			a.ExtendLo2ToInt64().StoreArray((*[2]int64)(out))
		})
	})
	t.Run("Int32x4/ExtendLo2ToInt64", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDIntegerResizeShape[int32, int64](t, "lo", 4, 2, func(x []int32, out []int64) {
			a := archsimd.LoadInt32x4Array((*[4]int32)(x))
			a.ExtendLo2ToInt64().StoreArray((*[2]int64)(out))
		})
	})
	t.Run("Uint8x16/ExtendLo2ToUint64", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDIntegerResizeShape[uint8, uint64](t, "lo", 16, 2, func(x []uint8, out []uint64) {
			a := archsimd.LoadUint8x16Array((*[16]uint8)(x))
			a.ExtendLo2ToUint64().StoreArray((*[2]uint64)(out))
		})
	})
	t.Run("Uint16x8/ExtendLo2ToUint64", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDIntegerResizeShape[uint16, uint64](t, "lo", 8, 2, func(x []uint16, out []uint64) {
			a := archsimd.LoadUint16x8Array((*[8]uint16)(x))
			a.ExtendLo2ToUint64().StoreArray((*[2]uint64)(out))
		})
	})
	t.Run("Uint32x4/ExtendLo2ToUint64", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDIntegerResizeShape[uint32, uint64](t, "lo", 4, 2, func(x []uint32, out []uint64) {
			a := archsimd.LoadUint32x4Array((*[4]uint32)(x))
			a.ExtendLo2ToUint64().StoreArray((*[2]uint64)(out))
		})
	})
	t.Run("Int8x16/ExtendLo4ToInt32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDIntegerResizeShape[int8, int32](t, "lo", 16, 4, func(x []int8, out []int32) {
			a := archsimd.LoadInt8x16Array((*[16]int8)(x))
			a.ExtendLo4ToInt32().StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Int16x8/ExtendLo4ToInt32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDIntegerResizeShape[int16, int32](t, "lo", 8, 4, func(x []int16, out []int32) {
			a := archsimd.LoadInt16x8Array((*[8]int16)(x))
			a.ExtendLo4ToInt32().StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Int8x16/ExtendLo4ToInt64", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX2() {
			t.Skip("official Go requires AVX2")
		}
		testSIMDIntegerResizeShape[int8, int64](t, "lo", 16, 4, func(x []int8, out []int64) {
			a := archsimd.LoadInt8x16Array((*[16]int8)(x))
			a.ExtendLo4ToInt64().StoreArray((*[4]int64)(out))
		})
	})
	t.Run("Int16x8/ExtendLo4ToInt64", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX2() {
			t.Skip("official Go requires AVX2")
		}
		testSIMDIntegerResizeShape[int16, int64](t, "lo", 8, 4, func(x []int16, out []int64) {
			a := archsimd.LoadInt16x8Array((*[8]int16)(x))
			a.ExtendLo4ToInt64().StoreArray((*[4]int64)(out))
		})
	})
	t.Run("Uint8x16/ExtendLo4ToUint32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDIntegerResizeShape[uint8, uint32](t, "lo", 16, 4, func(x []uint8, out []uint32) {
			a := archsimd.LoadUint8x16Array((*[16]uint8)(x))
			a.ExtendLo4ToUint32().StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Uint16x8/ExtendLo4ToUint32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDIntegerResizeShape[uint16, uint32](t, "lo", 8, 4, func(x []uint16, out []uint32) {
			a := archsimd.LoadUint16x8Array((*[8]uint16)(x))
			a.ExtendLo4ToUint32().StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Uint8x16/ExtendLo4ToUint64", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX2() {
			t.Skip("official Go requires AVX2")
		}
		testSIMDIntegerResizeShape[uint8, uint64](t, "lo", 16, 4, func(x []uint8, out []uint64) {
			a := archsimd.LoadUint8x16Array((*[16]uint8)(x))
			a.ExtendLo4ToUint64().StoreArray((*[4]uint64)(out))
		})
	})
	t.Run("Uint16x8/ExtendLo4ToUint64", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX2() {
			t.Skip("official Go requires AVX2")
		}
		testSIMDIntegerResizeShape[uint16, uint64](t, "lo", 8, 4, func(x []uint16, out []uint64) {
			a := archsimd.LoadUint16x8Array((*[8]uint16)(x))
			a.ExtendLo4ToUint64().StoreArray((*[4]uint64)(out))
		})
	})
	t.Run("Int8x16/ExtendLo8ToInt16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDIntegerResizeShape[int8, int16](t, "lo", 16, 8, func(x []int8, out []int16) {
			a := archsimd.LoadInt8x16Array((*[16]int8)(x))
			a.ExtendLo8ToInt16().StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int8x16/ExtendLo8ToInt32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX2() {
			t.Skip("official Go requires AVX2")
		}
		testSIMDIntegerResizeShape[int8, int32](t, "lo", 16, 8, func(x []int8, out []int32) {
			a := archsimd.LoadInt8x16Array((*[16]int8)(x))
			a.ExtendLo8ToInt32().StoreArray((*[8]int32)(out))
		})
	})
	t.Run("Uint8x16/ExtendLo8ToUint16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDIntegerResizeShape[uint8, uint16](t, "lo", 16, 8, func(x []uint8, out []uint16) {
			a := archsimd.LoadUint8x16Array((*[16]uint8)(x))
			a.ExtendLo8ToUint16().StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint8x16/ExtendLo8ToUint32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX2() {
			t.Skip("official Go requires AVX2")
		}
		testSIMDIntegerResizeShape[uint8, uint32](t, "lo", 16, 8, func(x []uint8, out []uint32) {
			a := archsimd.LoadUint8x16Array((*[16]uint8)(x))
			a.ExtendLo8ToUint32().StoreArray((*[8]uint32)(out))
		})
	})
	t.Run("Int8x16/ExtendToInt16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX2() {
			t.Skip("official Go requires AVX2")
		}
		testSIMDIntegerResizeShape[int8, int16](t, "extend", 16, 16, func(x []int8, out []int16) {
			a := archsimd.LoadInt8x16Array((*[16]int8)(x))
			a.ExtendToInt16().StoreArray((*[16]int16)(out))
		})
	})
	t.Run("Int16x8/ExtendToInt32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX2() {
			t.Skip("official Go requires AVX2")
		}
		testSIMDIntegerResizeShape[int16, int32](t, "extend", 8, 8, func(x []int16, out []int32) {
			a := archsimd.LoadInt16x8Array((*[8]int16)(x))
			a.ExtendToInt32().StoreArray((*[8]int32)(out))
		})
	})
	t.Run("Int32x4/ExtendToInt64", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX2() {
			t.Skip("official Go requires AVX2")
		}
		testSIMDIntegerResizeShape[int32, int64](t, "extend", 4, 4, func(x []int32, out []int64) {
			a := archsimd.LoadInt32x4Array((*[4]int32)(x))
			a.ExtendToInt64().StoreArray((*[4]int64)(out))
		})
	})
	t.Run("Uint8x16/ExtendToUint16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX2() {
			t.Skip("official Go requires AVX2")
		}
		testSIMDIntegerResizeShape[uint8, uint16](t, "extend", 16, 16, func(x []uint8, out []uint16) {
			a := archsimd.LoadUint8x16Array((*[16]uint8)(x))
			a.ExtendToUint16().StoreArray((*[16]uint16)(out))
		})
	})
	t.Run("Uint16x8/ExtendToUint32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX2() {
			t.Skip("official Go requires AVX2")
		}
		testSIMDIntegerResizeShape[uint16, uint32](t, "extend", 8, 8, func(x []uint16, out []uint32) {
			a := archsimd.LoadUint16x8Array((*[8]uint16)(x))
			a.ExtendToUint32().StoreArray((*[8]uint32)(out))
		})
	})
	t.Run("Uint32x4/ExtendToUint64", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX2() {
			t.Skip("official Go requires AVX2")
		}
		testSIMDIntegerResizeShape[uint32, uint64](t, "extend", 4, 4, func(x []uint32, out []uint64) {
			a := archsimd.LoadUint32x4Array((*[4]uint32)(x))
			a.ExtendToUint64().StoreArray((*[4]uint64)(out))
		})
	})
	t.Run("Float32x8/Floor", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDConversionShape[float32, float32](t, "Floor", 8, 8, func(x []float32, out []float32) {
			a := archsimd.LoadFloat32x8Array((*[8]float32)(x))
			a.Floor().StoreArray((*[8]float32)(out))
		})
	})
	t.Run("Float64x4/Floor", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDConversionShape[float64, float64](t, "Floor", 4, 4, func(x []float64, out []float64) {
			a := archsimd.LoadFloat64x4Array((*[4]float64)(x))
			a.Floor().StoreArray((*[4]float64)(out))
		})
	})
	t.Run("Float32x8/Round", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDConversionShape[float32, float32](t, "Round", 8, 8, func(x []float32, out []float32) {
			a := archsimd.LoadFloat32x8Array((*[8]float32)(x))
			a.Round().StoreArray((*[8]float32)(out))
		})
	})
	t.Run("Float64x4/Round", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDConversionShape[float64, float64](t, "Round", 4, 4, func(x []float64, out []float64) {
			a := archsimd.LoadFloat64x4Array((*[4]float64)(x))
			a.Round().StoreArray((*[4]float64)(out))
		})
	})
	t.Run("Int16x8/SaturateToInt8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int16, int8](t, "saturate", 8, 16, func(x []int16, out []int8) {
			a := archsimd.LoadInt16x8Array((*[8]int16)(x))
			a.SaturateToInt8().StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int16x16/SaturateToInt8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int16, int8](t, "saturate", 16, 16, func(x []int16, out []int8) {
			a := archsimd.LoadInt16x16Array((*[16]int16)(x))
			a.SaturateToInt8().StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int32x4/SaturateToInt8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int32, int8](t, "saturate", 4, 16, func(x []int32, out []int8) {
			a := archsimd.LoadInt32x4Array((*[4]int32)(x))
			a.SaturateToInt8().StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int32x8/SaturateToInt8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int32, int8](t, "saturate", 8, 16, func(x []int32, out []int8) {
			a := archsimd.LoadInt32x8Array((*[8]int32)(x))
			a.SaturateToInt8().StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int64x2/SaturateToInt8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int64, int8](t, "saturate", 2, 16, func(x []int64, out []int8) {
			a := archsimd.LoadInt64x2Array((*[2]int64)(x))
			a.SaturateToInt8().StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int64x4/SaturateToInt8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int64, int8](t, "saturate", 4, 16, func(x []int64, out []int8) {
			a := archsimd.LoadInt64x4Array((*[4]int64)(x))
			a.SaturateToInt8().StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int32x4/SaturateToInt16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int32, int16](t, "saturate", 4, 8, func(x []int32, out []int16) {
			a := archsimd.LoadInt32x4Array((*[4]int32)(x))
			a.SaturateToInt16().StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int32x8/SaturateToInt16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int32, int16](t, "saturate", 8, 8, func(x []int32, out []int16) {
			a := archsimd.LoadInt32x8Array((*[8]int32)(x))
			a.SaturateToInt16().StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int64x2/SaturateToInt16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int64, int16](t, "saturate", 2, 8, func(x []int64, out []int16) {
			a := archsimd.LoadInt64x2Array((*[2]int64)(x))
			a.SaturateToInt16().StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int64x4/SaturateToInt16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int64, int16](t, "saturate", 4, 8, func(x []int64, out []int16) {
			a := archsimd.LoadInt64x4Array((*[4]int64)(x))
			a.SaturateToInt16().StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int32x4/SaturateToInt16Concat", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDIntegerResizeShape[int32, int16](t, "concat", 4, 8, func(x []int32, out []int16) {
			a := archsimd.LoadInt32x4Array((*[4]int32)(x))
			b := archsimd.LoadInt32x4Array((*[4]int32)(x[4:]))
			a.SaturateToInt16Concat(b).StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int32x8/SaturateToInt16ConcatGrouped", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX2() {
			t.Skip("official Go requires AVX2")
		}
		testSIMDIntegerResizeShape[int32, int16](t, "grouped", 8, 16, func(x []int32, out []int16) {
			a := archsimd.LoadInt32x8Array((*[8]int32)(x))
			b := archsimd.LoadInt32x8Array((*[8]int32)(x[8:]))
			a.SaturateToInt16ConcatGrouped(b).StoreArray((*[16]int16)(out))
		})
	})
	t.Run("Int64x2/SaturateToInt32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int64, int32](t, "saturate", 2, 4, func(x []int64, out []int32) {
			a := archsimd.LoadInt64x2Array((*[2]int64)(x))
			a.SaturateToInt32().StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Int64x4/SaturateToInt32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int64, int32](t, "saturate", 4, 4, func(x []int64, out []int32) {
			a := archsimd.LoadInt64x4Array((*[4]int64)(x))
			a.SaturateToInt32().StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Uint16x8/SaturateToUint8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint16, uint8](t, "saturate", 8, 16, func(x []uint16, out []uint8) {
			a := archsimd.LoadUint16x8Array((*[8]uint16)(x))
			a.SaturateToUint8().StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint16x16/SaturateToUint8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint16, uint8](t, "saturate", 16, 16, func(x []uint16, out []uint8) {
			a := archsimd.LoadUint16x16Array((*[16]uint16)(x))
			a.SaturateToUint8().StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint32x4/SaturateToUint8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint32, uint8](t, "saturate", 4, 16, func(x []uint32, out []uint8) {
			a := archsimd.LoadUint32x4Array((*[4]uint32)(x))
			a.SaturateToUint8().StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint32x8/SaturateToUint8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint32, uint8](t, "saturate", 8, 16, func(x []uint32, out []uint8) {
			a := archsimd.LoadUint32x8Array((*[8]uint32)(x))
			a.SaturateToUint8().StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint64x2/SaturateToUint8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint64, uint8](t, "saturate", 2, 16, func(x []uint64, out []uint8) {
			a := archsimd.LoadUint64x2Array((*[2]uint64)(x))
			a.SaturateToUint8().StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint64x4/SaturateToUint8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint64, uint8](t, "saturate", 4, 16, func(x []uint64, out []uint8) {
			a := archsimd.LoadUint64x4Array((*[4]uint64)(x))
			a.SaturateToUint8().StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint32x4/SaturateToUint16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint32, uint16](t, "saturate", 4, 8, func(x []uint32, out []uint16) {
			a := archsimd.LoadUint32x4Array((*[4]uint32)(x))
			a.SaturateToUint16().StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint32x8/SaturateToUint16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint32, uint16](t, "saturate", 8, 8, func(x []uint32, out []uint16) {
			a := archsimd.LoadUint32x8Array((*[8]uint32)(x))
			a.SaturateToUint16().StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint64x2/SaturateToUint16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint64, uint16](t, "saturate", 2, 8, func(x []uint64, out []uint16) {
			a := archsimd.LoadUint64x2Array((*[2]uint64)(x))
			a.SaturateToUint16().StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint64x4/SaturateToUint16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint64, uint16](t, "saturate", 4, 8, func(x []uint64, out []uint16) {
			a := archsimd.LoadUint64x4Array((*[4]uint64)(x))
			a.SaturateToUint16().StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Int32x4/SaturateToUint16Concat", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDIntegerResizeShape[int32, uint16](t, "concat", 4, 8, func(x []int32, out []uint16) {
			a := archsimd.LoadInt32x4Array((*[4]int32)(x))
			b := archsimd.LoadInt32x4Array((*[4]int32)(x[4:]))
			a.SaturateToUint16Concat(b).StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Int32x8/SaturateToUint16ConcatGrouped", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX2() {
			t.Skip("official Go requires AVX2")
		}
		testSIMDIntegerResizeShape[int32, uint16](t, "grouped", 8, 16, func(x []int32, out []uint16) {
			a := archsimd.LoadInt32x8Array((*[8]int32)(x))
			b := archsimd.LoadInt32x8Array((*[8]int32)(x[8:]))
			a.SaturateToUint16ConcatGrouped(b).StoreArray((*[16]uint16)(out))
		})
	})
	t.Run("Uint64x2/SaturateToUint32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint64, uint32](t, "saturate", 2, 4, func(x []uint64, out []uint32) {
			a := archsimd.LoadUint64x2Array((*[2]uint64)(x))
			a.SaturateToUint32().StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Uint64x4/SaturateToUint32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint64, uint32](t, "saturate", 4, 4, func(x []uint64, out []uint32) {
			a := archsimd.LoadUint64x4Array((*[4]uint64)(x))
			a.SaturateToUint32().StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Float32x8/Trunc", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDConversionShape[float32, float32](t, "Trunc", 8, 8, func(x []float32, out []float32) {
			a := archsimd.LoadFloat32x8Array((*[8]float32)(x))
			a.Trunc().StoreArray((*[8]float32)(out))
		})
	})
	t.Run("Float64x4/Trunc", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
			t.Skip("official Go requires AVX")
		}
		testSIMDConversionShape[float64, float64](t, "Trunc", 4, 4, func(x []float64, out []float64) {
			a := archsimd.LoadFloat64x4Array((*[4]float64)(x))
			a.Trunc().StoreArray((*[4]float64)(out))
		})
	})
	t.Run("Int16x8/TruncToInt8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int16, int8](t, "truncate", 8, 16, func(x []int16, out []int8) {
			a := archsimd.LoadInt16x8Array((*[8]int16)(x))
			a.TruncToInt8().StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int16x16/TruncToInt8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int16, int8](t, "truncate", 16, 16, func(x []int16, out []int8) {
			a := archsimd.LoadInt16x16Array((*[16]int16)(x))
			a.TruncToInt8().StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int32x4/TruncToInt8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int32, int8](t, "truncate", 4, 16, func(x []int32, out []int8) {
			a := archsimd.LoadInt32x4Array((*[4]int32)(x))
			a.TruncToInt8().StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int32x8/TruncToInt8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int32, int8](t, "truncate", 8, 16, func(x []int32, out []int8) {
			a := archsimd.LoadInt32x8Array((*[8]int32)(x))
			a.TruncToInt8().StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int64x2/TruncToInt8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int64, int8](t, "truncate", 2, 16, func(x []int64, out []int8) {
			a := archsimd.LoadInt64x2Array((*[2]int64)(x))
			a.TruncToInt8().StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int64x4/TruncToInt8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int64, int8](t, "truncate", 4, 16, func(x []int64, out []int8) {
			a := archsimd.LoadInt64x4Array((*[4]int64)(x))
			a.TruncToInt8().StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int32x4/TruncToInt16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int32, int16](t, "truncate", 4, 8, func(x []int32, out []int16) {
			a := archsimd.LoadInt32x4Array((*[4]int32)(x))
			a.TruncToInt16().StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int32x8/TruncToInt16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int32, int16](t, "truncate", 8, 8, func(x []int32, out []int16) {
			a := archsimd.LoadInt32x8Array((*[8]int32)(x))
			a.TruncToInt16().StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int64x2/TruncToInt16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int64, int16](t, "truncate", 2, 8, func(x []int64, out []int16) {
			a := archsimd.LoadInt64x2Array((*[2]int64)(x))
			a.TruncToInt16().StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int64x4/TruncToInt16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int64, int16](t, "truncate", 4, 8, func(x []int64, out []int16) {
			a := archsimd.LoadInt64x4Array((*[4]int64)(x))
			a.TruncToInt16().StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int64x2/TruncToInt32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int64, int32](t, "truncate", 2, 4, func(x []int64, out []int32) {
			a := archsimd.LoadInt64x2Array((*[2]int64)(x))
			a.TruncToInt32().StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Int64x4/TruncToInt32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[int64, int32](t, "truncate", 4, 4, func(x []int64, out []int32) {
			a := archsimd.LoadInt64x4Array((*[4]int64)(x))
			a.TruncToInt32().StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Uint16x8/TruncToUint8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint16, uint8](t, "truncate", 8, 16, func(x []uint16, out []uint8) {
			a := archsimd.LoadUint16x8Array((*[8]uint16)(x))
			a.TruncToUint8().StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint16x16/TruncToUint8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint16, uint8](t, "truncate", 16, 16, func(x []uint16, out []uint8) {
			a := archsimd.LoadUint16x16Array((*[16]uint16)(x))
			a.TruncToUint8().StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint32x4/TruncToUint8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint32, uint8](t, "truncate", 4, 16, func(x []uint32, out []uint8) {
			a := archsimd.LoadUint32x4Array((*[4]uint32)(x))
			a.TruncToUint8().StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint32x8/TruncToUint8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint32, uint8](t, "truncate", 8, 16, func(x []uint32, out []uint8) {
			a := archsimd.LoadUint32x8Array((*[8]uint32)(x))
			a.TruncToUint8().StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint64x2/TruncToUint8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint64, uint8](t, "truncate", 2, 16, func(x []uint64, out []uint8) {
			a := archsimd.LoadUint64x2Array((*[2]uint64)(x))
			a.TruncToUint8().StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint64x4/TruncToUint8", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint64, uint8](t, "truncate", 4, 16, func(x []uint64, out []uint8) {
			a := archsimd.LoadUint64x4Array((*[4]uint64)(x))
			a.TruncToUint8().StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint32x4/TruncToUint16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint32, uint16](t, "truncate", 4, 8, func(x []uint32, out []uint16) {
			a := archsimd.LoadUint32x4Array((*[4]uint32)(x))
			a.TruncToUint16().StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint32x8/TruncToUint16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint32, uint16](t, "truncate", 8, 8, func(x []uint32, out []uint16) {
			a := archsimd.LoadUint32x8Array((*[8]uint32)(x))
			a.TruncToUint16().StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint64x2/TruncToUint16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint64, uint16](t, "truncate", 2, 8, func(x []uint64, out []uint16) {
			a := archsimd.LoadUint64x2Array((*[2]uint64)(x))
			a.TruncToUint16().StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint64x4/TruncToUint16", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint64, uint16](t, "truncate", 4, 8, func(x []uint64, out []uint16) {
			a := archsimd.LoadUint64x4Array((*[4]uint64)(x))
			a.TruncToUint16().StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint64x2/TruncToUint32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint64, uint32](t, "truncate", 2, 4, func(x []uint64, out []uint32) {
			a := archsimd.LoadUint64x2Array((*[2]uint64)(x))
			a.TruncToUint32().StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Uint64x4/TruncToUint32", func(t *testing.T) {
		if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
			t.Skip("official Go requires AVX512")
		}
		testSIMDIntegerResizeShape[uint64, uint32](t, "truncate", 4, 4, func(x []uint64, out []uint32) {
			a := archsimd.LoadUint64x4Array((*[4]uint64)(x))
			a.TruncToUint32().StoreArray((*[4]uint32)(out))
		})
	})
}

var conversionWidenIndirect = archsimd.Float32x4.ConvertToFloat64
var conversionNarrowIndirect = archsimd.Float64x4.ConvertToFloat32
var resizeWidenIndirect = archsimd.Int8x16.ExtendToInt16
var resizeNarrowIndirect = archsimd.Int32x8.SaturateToInt16ConcatGrouped

func TestSIMDConversionResizeIndirect(t *testing.T) {
	requireSIMD256(t)
	testSIMDConversionShape[float32, float64](t, "ConvertToFloat64", 4, 4, func(x []float32, out []float64) {
		conversionWidenIndirect(archsimd.LoadFloat32x4Array((*[4]float32)(x))).StoreArray((*[4]float64)(out))
	})
	testSIMDConversionShape[float64, float32](t, "ConvertToFloat32", 4, 4, func(x []float64, out []float32) {
		conversionNarrowIndirect(archsimd.LoadFloat64x4Array((*[4]float64)(x))).StoreArray((*[4]float32)(out))
	})
	testSIMDIntegerResizeShape[int8, int16](t, "extend", 16, 16, func(x []int8, out []int16) {
		resizeWidenIndirect(archsimd.LoadInt8x16Array((*[16]int8)(x))).StoreArray((*[16]int16)(out))
	})
	testSIMDIntegerResizeShape[int32, int16](t, "grouped", 8, 16, func(x []int32, out []int16) {
		a, b := archsimd.LoadInt32x8Array((*[8]int32)(x)), archsimd.LoadInt32x8Array((*[8]int32)(x[8:]))
		resizeNarrowIndirect(a, b).StoreArray((*[16]int16)(out))
	})
}

//go:noinline
func conversionWidenGuarded(x archsimd.Float32x4) archsimd.Float64x4 {
	if archsimd.X86.AVX2() {
		return x.ConvertToFloat64()
	}
	return conversionWidenIndirect(x)
}

func TestSIMDConversionResizeFMV(t *testing.T) {
	requireSIMD256(t)
	testSIMDConversionShape[float32, float64](t, "ConvertToFloat64", 4, 4, func(x []float32, out []float64) {
		conversionWidenGuarded(archsimd.LoadFloat32x4Array((*[4]float32)(x))).StoreArray((*[4]float64)(out))
	})
}
