//go:build goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDHorizontalSub(t *testing.T) {
	requireHorizontal128(t)
	t.Run("Float32x4", func(t *testing.T) {
		testHorizontalSub[float32](t, 4, 4, func(x, y, out []float32) {
			a, b := archsimd.LoadFloat32x4Array((*[4]float32)(x)), archsimd.LoadFloat32x4Array((*[4]float32)(y))
			a.ConcatSubPairs(b).StoreArray((*[4]float32)(out))
		})
	})
	t.Run("Float64x2", func(t *testing.T) {
		testHorizontalSub[float64](t, 2, 2, func(x, y, out []float64) {
			a, b := archsimd.LoadFloat64x2Array((*[2]float64)(x)), archsimd.LoadFloat64x2Array((*[2]float64)(y))
			a.ConcatSubPairs(b).StoreArray((*[2]float64)(out))
		})
	})
	t.Run("Int16x8", func(t *testing.T) {
		testHorizontalSub[int16](t, 8, 8, func(x, y, out []int16) {
			a, b := archsimd.LoadInt16x8Array((*[8]int16)(x)), archsimd.LoadInt16x8Array((*[8]int16)(y))
			a.ConcatSubPairs(b).StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int32x4", func(t *testing.T) {
		testHorizontalSub[int32](t, 4, 4, func(x, y, out []int32) {
			a, b := archsimd.LoadInt32x4Array((*[4]int32)(x)), archsimd.LoadInt32x4Array((*[4]int32)(y))
			a.ConcatSubPairs(b).StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Uint16x8", func(t *testing.T) {
		testHorizontalSub[uint16](t, 8, 8, func(x, y, out []uint16) {
			a, b := archsimd.LoadUint16x8Array((*[8]uint16)(x)), archsimd.LoadUint16x8Array((*[8]uint16)(y))
			a.ConcatSubPairs(b).StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint32x4", func(t *testing.T) {
		testHorizontalSub[uint32](t, 4, 4, func(x, y, out []uint32) {
			a, b := archsimd.LoadUint32x4Array((*[4]uint32)(x)), archsimd.LoadUint32x4Array((*[4]uint32)(y))
			a.ConcatSubPairs(b).StoreArray((*[4]uint32)(out))
		})
	})
}

func TestSIMD256HorizontalSub(t *testing.T) {
	requireSIMD256(t)
	t.Run("Float32x8", func(t *testing.T) {
		testHorizontalSub[float32](t, 8, 4, func(x, y, out []float32) {
			a, b := archsimd.LoadFloat32x8Array((*[8]float32)(x)), archsimd.LoadFloat32x8Array((*[8]float32)(y))
			a.ConcatSubPairsGrouped(b).StoreArray((*[8]float32)(out))
		})
	})
	t.Run("Float64x4", func(t *testing.T) {
		testHorizontalSub[float64](t, 4, 2, func(x, y, out []float64) {
			a, b := archsimd.LoadFloat64x4Array((*[4]float64)(x)), archsimd.LoadFloat64x4Array((*[4]float64)(y))
			a.ConcatSubPairsGrouped(b).StoreArray((*[4]float64)(out))
		})
	})
	t.Run("Int16x16", func(t *testing.T) {
		testHorizontalSub[int16](t, 16, 8, func(x, y, out []int16) {
			a, b := archsimd.LoadInt16x16Array((*[16]int16)(x)), archsimd.LoadInt16x16Array((*[16]int16)(y))
			a.ConcatSubPairsGrouped(b).StoreArray((*[16]int16)(out))
		})
	})
	t.Run("Int32x8", func(t *testing.T) {
		testHorizontalSub[int32](t, 8, 4, func(x, y, out []int32) {
			a, b := archsimd.LoadInt32x8Array((*[8]int32)(x)), archsimd.LoadInt32x8Array((*[8]int32)(y))
			a.ConcatSubPairsGrouped(b).StoreArray((*[8]int32)(out))
		})
	})
	t.Run("Uint16x16", func(t *testing.T) {
		testHorizontalSub[uint16](t, 16, 8, func(x, y, out []uint16) {
			a, b := archsimd.LoadUint16x16Array((*[16]uint16)(x)), archsimd.LoadUint16x16Array((*[16]uint16)(y))
			a.ConcatSubPairsGrouped(b).StoreArray((*[16]uint16)(out))
		})
	})
	t.Run("Uint32x8", func(t *testing.T) {
		testHorizontalSub[uint32](t, 8, 4, func(x, y, out []uint32) {
			a, b := archsimd.LoadUint32x8Array((*[8]uint32)(x)), archsimd.LoadUint32x8Array((*[8]uint32)(y))
			a.ConcatSubPairsGrouped(b).StoreArray((*[8]uint32)(out))
		})
	})
}

// Keep the method expression mutable so calls exercise the intrinsic definition
// and feature-independent wide ABI, in addition to direct lowering above.
var horizontalSubIndirect = archsimd.Float32x8.ConcatSubPairsGrouped

func TestSIMD256HorizontalSubIndirect(t *testing.T) {
	requireSIMD256(t)
	testHorizontalSub[float32](t, 8, 4, func(x, y, out []float32) {
		a, b := archsimd.LoadFloat32x8Array((*[8]float32)(x)), archsimd.LoadFloat32x8Array((*[8]float32)(y))
		horizontalSubIndirect(a, b).StoreArray((*[8]float32)(out))
	})
}

//go:noinline
func horizontalSubGroupedBaseline(x, y archsimd.Float32x8) archsimd.Float32x8 {
	return x.ConcatSubPairsGrouped(y)
}

//go:noinline
func horizontalSubGroupedGuarded(x, y archsimd.Float32x8) archsimd.Float32x8 {
	if archsimd.X86.AVX2() {
		return x.ConcatSubPairsGrouped(y)
	}
	return horizontalSubGroupedBaseline(x, y)
}

func TestSIMD256HorizontalSubFMV(t *testing.T) {
	requireSIMD256(t)
	testHorizontalSub[float32](t, 8, 4, func(x, y, out []float32) {
		a, b := archsimd.LoadFloat32x8Array((*[8]float32)(x)), archsimd.LoadFloat32x8Array((*[8]float32)(y))
		horizontalSubGroupedGuarded(a, b).StoreArray((*[8]float32)(out))
	})
}
