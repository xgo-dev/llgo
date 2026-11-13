//go:build goexperiment.simd && amd64

package simd_test

import (
	"runtime"
	"simd/archsimd"
	"testing"
)

func TestSIMD256HorizontalAdd(t *testing.T) {
	requireSIMD256(t)
	t.Run("Float32x8", func(t *testing.T) {
		testHorizontal[float32](t, 8, 4, func(x, y, out []float32) {
			a, b := archsimd.LoadFloat32x8Array((*[8]float32)(x)), archsimd.LoadFloat32x8Array((*[8]float32)(y))
			a.ConcatAddPairsGrouped(b).StoreArray((*[8]float32)(out))
		})
	})
	t.Run("Float64x4", func(t *testing.T) {
		testHorizontal[float64](t, 4, 2, func(x, y, out []float64) {
			a, b := archsimd.LoadFloat64x4Array((*[4]float64)(x)), archsimd.LoadFloat64x4Array((*[4]float64)(y))
			a.ConcatAddPairsGrouped(b).StoreArray((*[4]float64)(out))
		})
	})
	t.Run("Int16x16", func(t *testing.T) {
		testHorizontal[int16](t, 16, 8, func(x, y, out []int16) {
			a, b := archsimd.LoadInt16x16Array((*[16]int16)(x)), archsimd.LoadInt16x16Array((*[16]int16)(y))
			a.ConcatAddPairsGrouped(b).StoreArray((*[16]int16)(out))
		})
	})
	t.Run("Int32x8", func(t *testing.T) {
		testHorizontal[int32](t, 8, 4, func(x, y, out []int32) {
			a, b := archsimd.LoadInt32x8Array((*[8]int32)(x)), archsimd.LoadInt32x8Array((*[8]int32)(y))
			a.ConcatAddPairsGrouped(b).StoreArray((*[8]int32)(out))
		})
	})
	t.Run("Uint16x16", func(t *testing.T) {
		testHorizontal[uint16](t, 16, 8, func(x, y, out []uint16) {
			a, b := archsimd.LoadUint16x16Array((*[16]uint16)(x)), archsimd.LoadUint16x16Array((*[16]uint16)(y))
			a.ConcatAddPairsGrouped(b).StoreArray((*[16]uint16)(out))
		})
	})
	t.Run("Uint32x8", func(t *testing.T) {
		testHorizontal[uint32](t, 8, 4, func(x, y, out []uint32) {
			a, b := archsimd.LoadUint32x8Array((*[8]uint32)(x)), archsimd.LoadUint32x8Array((*[8]uint32)(y))
			a.ConcatAddPairsGrouped(b).StoreArray((*[8]uint32)(out))
		})
	})
}

func requireHorizontal128(t *testing.T) {
	t.Helper()
	if runtime.Compiler != "llgo" && !archsimd.X86.AVX() {
		t.Skip("official Go horizontal add requires AVX")
	}
}

// Keep the method expression mutable so calls exercise the intrinsic definition
// and feature-independent wide ABI, in addition to direct lowering above.
var horizontalAddIndirect = archsimd.Float32x8.ConcatAddPairsGrouped

func TestSIMD256HorizontalIndirect(t *testing.T) {
	requireSIMD256(t)
	testHorizontal[float32](t, 8, 4, func(x, y, out []float32) {
		a, b := archsimd.LoadFloat32x8Array((*[8]float32)(x)), archsimd.LoadFloat32x8Array((*[8]float32)(y))
		horizontalAddIndirect(a, b).StoreArray((*[8]float32)(out))
	})
}

//go:noinline
func horizontalGroupedBaseline(x, y archsimd.Float32x8) archsimd.Float32x8 {
	return x.ConcatAddPairsGrouped(y)
}

//go:noinline
func horizontalGroupedGuarded(x, y archsimd.Float32x8) archsimd.Float32x8 {
	if archsimd.X86.AVX2() {
		return x.ConcatAddPairsGrouped(y)
	}
	return horizontalGroupedBaseline(x, y)
}

func TestSIMD256HorizontalFMV(t *testing.T) {
	requireSIMD256(t)
	testHorizontal[float32](t, 8, 4, func(x, y, out []float32) {
		a, b := archsimd.LoadFloat32x8Array((*[8]float32)(x)), archsimd.LoadFloat32x8Array((*[8]float32)(y))
		horizontalGroupedGuarded(a, b).StoreArray((*[8]float32)(out))
	})
}
