//go:build goexperiment.simd && (amd64 || arm64)

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDHorizontalAdd(t *testing.T) {
	requireHorizontal128(t)
	t.Run("Float32x4", func(t *testing.T) {
		testHorizontal[float32](t, 4, 4, func(x, y, out []float32) {
			a, b := archsimd.LoadFloat32x4Array((*[4]float32)(x)), archsimd.LoadFloat32x4Array((*[4]float32)(y))
			a.ConcatAddPairs(b).StoreArray((*[4]float32)(out))
		})
	})
	t.Run("Float64x2", func(t *testing.T) {
		testHorizontal[float64](t, 2, 2, func(x, y, out []float64) {
			a, b := archsimd.LoadFloat64x2Array((*[2]float64)(x)), archsimd.LoadFloat64x2Array((*[2]float64)(y))
			a.ConcatAddPairs(b).StoreArray((*[2]float64)(out))
		})
	})
	t.Run("Int16x8", func(t *testing.T) {
		testHorizontal[int16](t, 8, 8, func(x, y, out []int16) {
			a, b := archsimd.LoadInt16x8Array((*[8]int16)(x)), archsimd.LoadInt16x8Array((*[8]int16)(y))
			a.ConcatAddPairs(b).StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int32x4", func(t *testing.T) {
		testHorizontal[int32](t, 4, 4, func(x, y, out []int32) {
			a, b := archsimd.LoadInt32x4Array((*[4]int32)(x)), archsimd.LoadInt32x4Array((*[4]int32)(y))
			a.ConcatAddPairs(b).StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Uint16x8", func(t *testing.T) {
		testHorizontal[uint16](t, 8, 8, func(x, y, out []uint16) {
			a, b := archsimd.LoadUint16x8Array((*[8]uint16)(x)), archsimd.LoadUint16x8Array((*[8]uint16)(y))
			a.ConcatAddPairs(b).StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint32x4", func(t *testing.T) {
		testHorizontal[uint32](t, 4, 4, func(x, y, out []uint32) {
			a, b := archsimd.LoadUint32x4Array((*[4]uint32)(x)), archsimd.LoadUint32x4Array((*[4]uint32)(y))
			a.ConcatAddPairs(b).StoreArray((*[4]uint32)(out))
		})
	})
}
