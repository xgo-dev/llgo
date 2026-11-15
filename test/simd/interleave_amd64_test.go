//go:build goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDInterleave(t *testing.T) {
	requireHorizontal128(t)
	t.Run("Int16x8/InterleaveHi", func(t *testing.T) {
		testInterleave[int16](t, 8, 8, "InterleaveHi", func(x, y, out []int16) {
			a, b := archsimd.LoadInt16x8Array((*[8]int16)(x)), archsimd.LoadInt16x8Array((*[8]int16)(y))
			a.InterleaveHi(b).StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int32x4/InterleaveHi", func(t *testing.T) {
		testInterleave[int32](t, 4, 4, "InterleaveHi", func(x, y, out []int32) {
			a, b := archsimd.LoadInt32x4Array((*[4]int32)(x)), archsimd.LoadInt32x4Array((*[4]int32)(y))
			a.InterleaveHi(b).StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Int64x2/InterleaveHi", func(t *testing.T) {
		testInterleave[int64](t, 2, 2, "InterleaveHi", func(x, y, out []int64) {
			a, b := archsimd.LoadInt64x2Array((*[2]int64)(x)), archsimd.LoadInt64x2Array((*[2]int64)(y))
			a.InterleaveHi(b).StoreArray((*[2]int64)(out))
		})
	})
	t.Run("Uint16x8/InterleaveHi", func(t *testing.T) {
		testInterleave[uint16](t, 8, 8, "InterleaveHi", func(x, y, out []uint16) {
			a, b := archsimd.LoadUint16x8Array((*[8]uint16)(x)), archsimd.LoadUint16x8Array((*[8]uint16)(y))
			a.InterleaveHi(b).StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint32x4/InterleaveHi", func(t *testing.T) {
		testInterleave[uint32](t, 4, 4, "InterleaveHi", func(x, y, out []uint32) {
			a, b := archsimd.LoadUint32x4Array((*[4]uint32)(x)), archsimd.LoadUint32x4Array((*[4]uint32)(y))
			a.InterleaveHi(b).StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Uint64x2/InterleaveHi", func(t *testing.T) {
		testInterleave[uint64](t, 2, 2, "InterleaveHi", func(x, y, out []uint64) {
			a, b := archsimd.LoadUint64x2Array((*[2]uint64)(x)), archsimd.LoadUint64x2Array((*[2]uint64)(y))
			a.InterleaveHi(b).StoreArray((*[2]uint64)(out))
		})
	})
	t.Run("Int16x16/InterleaveHiGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testInterleave[int16](t, 16, 8, "InterleaveHiGrouped", func(x, y, out []int16) {
			a, b := archsimd.LoadInt16x16Array((*[16]int16)(x)), archsimd.LoadInt16x16Array((*[16]int16)(y))
			a.InterleaveHiGrouped(b).StoreArray((*[16]int16)(out))
		})
	})
	t.Run("Int32x8/InterleaveHiGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testInterleave[int32](t, 8, 4, "InterleaveHiGrouped", func(x, y, out []int32) {
			a, b := archsimd.LoadInt32x8Array((*[8]int32)(x)), archsimd.LoadInt32x8Array((*[8]int32)(y))
			a.InterleaveHiGrouped(b).StoreArray((*[8]int32)(out))
		})
	})
	t.Run("Int64x4/InterleaveHiGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testInterleave[int64](t, 4, 2, "InterleaveHiGrouped", func(x, y, out []int64) {
			a, b := archsimd.LoadInt64x4Array((*[4]int64)(x)), archsimd.LoadInt64x4Array((*[4]int64)(y))
			a.InterleaveHiGrouped(b).StoreArray((*[4]int64)(out))
		})
	})
	t.Run("Uint16x16/InterleaveHiGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testInterleave[uint16](t, 16, 8, "InterleaveHiGrouped", func(x, y, out []uint16) {
			a, b := archsimd.LoadUint16x16Array((*[16]uint16)(x)), archsimd.LoadUint16x16Array((*[16]uint16)(y))
			a.InterleaveHiGrouped(b).StoreArray((*[16]uint16)(out))
		})
	})
	t.Run("Uint32x8/InterleaveHiGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testInterleave[uint32](t, 8, 4, "InterleaveHiGrouped", func(x, y, out []uint32) {
			a, b := archsimd.LoadUint32x8Array((*[8]uint32)(x)), archsimd.LoadUint32x8Array((*[8]uint32)(y))
			a.InterleaveHiGrouped(b).StoreArray((*[8]uint32)(out))
		})
	})
	t.Run("Uint64x4/InterleaveHiGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testInterleave[uint64](t, 4, 2, "InterleaveHiGrouped", func(x, y, out []uint64) {
			a, b := archsimd.LoadUint64x4Array((*[4]uint64)(x)), archsimd.LoadUint64x4Array((*[4]uint64)(y))
			a.InterleaveHiGrouped(b).StoreArray((*[4]uint64)(out))
		})
	})
	t.Run("Int16x8/InterleaveLo", func(t *testing.T) {
		testInterleave[int16](t, 8, 8, "InterleaveLo", func(x, y, out []int16) {
			a, b := archsimd.LoadInt16x8Array((*[8]int16)(x)), archsimd.LoadInt16x8Array((*[8]int16)(y))
			a.InterleaveLo(b).StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int32x4/InterleaveLo", func(t *testing.T) {
		testInterleave[int32](t, 4, 4, "InterleaveLo", func(x, y, out []int32) {
			a, b := archsimd.LoadInt32x4Array((*[4]int32)(x)), archsimd.LoadInt32x4Array((*[4]int32)(y))
			a.InterleaveLo(b).StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Int64x2/InterleaveLo", func(t *testing.T) {
		testInterleave[int64](t, 2, 2, "InterleaveLo", func(x, y, out []int64) {
			a, b := archsimd.LoadInt64x2Array((*[2]int64)(x)), archsimd.LoadInt64x2Array((*[2]int64)(y))
			a.InterleaveLo(b).StoreArray((*[2]int64)(out))
		})
	})
	t.Run("Uint16x8/InterleaveLo", func(t *testing.T) {
		testInterleave[uint16](t, 8, 8, "InterleaveLo", func(x, y, out []uint16) {
			a, b := archsimd.LoadUint16x8Array((*[8]uint16)(x)), archsimd.LoadUint16x8Array((*[8]uint16)(y))
			a.InterleaveLo(b).StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint32x4/InterleaveLo", func(t *testing.T) {
		testInterleave[uint32](t, 4, 4, "InterleaveLo", func(x, y, out []uint32) {
			a, b := archsimd.LoadUint32x4Array((*[4]uint32)(x)), archsimd.LoadUint32x4Array((*[4]uint32)(y))
			a.InterleaveLo(b).StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Uint64x2/InterleaveLo", func(t *testing.T) {
		testInterleave[uint64](t, 2, 2, "InterleaveLo", func(x, y, out []uint64) {
			a, b := archsimd.LoadUint64x2Array((*[2]uint64)(x)), archsimd.LoadUint64x2Array((*[2]uint64)(y))
			a.InterleaveLo(b).StoreArray((*[2]uint64)(out))
		})
	})
	t.Run("Int16x16/InterleaveLoGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testInterleave[int16](t, 16, 8, "InterleaveLoGrouped", func(x, y, out []int16) {
			a, b := archsimd.LoadInt16x16Array((*[16]int16)(x)), archsimd.LoadInt16x16Array((*[16]int16)(y))
			a.InterleaveLoGrouped(b).StoreArray((*[16]int16)(out))
		})
	})
	t.Run("Int32x8/InterleaveLoGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testInterleave[int32](t, 8, 4, "InterleaveLoGrouped", func(x, y, out []int32) {
			a, b := archsimd.LoadInt32x8Array((*[8]int32)(x)), archsimd.LoadInt32x8Array((*[8]int32)(y))
			a.InterleaveLoGrouped(b).StoreArray((*[8]int32)(out))
		})
	})
	t.Run("Int64x4/InterleaveLoGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testInterleave[int64](t, 4, 2, "InterleaveLoGrouped", func(x, y, out []int64) {
			a, b := archsimd.LoadInt64x4Array((*[4]int64)(x)), archsimd.LoadInt64x4Array((*[4]int64)(y))
			a.InterleaveLoGrouped(b).StoreArray((*[4]int64)(out))
		})
	})
	t.Run("Uint16x16/InterleaveLoGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testInterleave[uint16](t, 16, 8, "InterleaveLoGrouped", func(x, y, out []uint16) {
			a, b := archsimd.LoadUint16x16Array((*[16]uint16)(x)), archsimd.LoadUint16x16Array((*[16]uint16)(y))
			a.InterleaveLoGrouped(b).StoreArray((*[16]uint16)(out))
		})
	})
	t.Run("Uint32x8/InterleaveLoGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testInterleave[uint32](t, 8, 4, "InterleaveLoGrouped", func(x, y, out []uint32) {
			a, b := archsimd.LoadUint32x8Array((*[8]uint32)(x)), archsimd.LoadUint32x8Array((*[8]uint32)(y))
			a.InterleaveLoGrouped(b).StoreArray((*[8]uint32)(out))
		})
	})
	t.Run("Uint64x4/InterleaveLoGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testInterleave[uint64](t, 4, 2, "InterleaveLoGrouped", func(x, y, out []uint64) {
			a, b := archsimd.LoadUint64x4Array((*[4]uint64)(x)), archsimd.LoadUint64x4Array((*[4]uint64)(y))
			a.InterleaveLoGrouped(b).StoreArray((*[4]uint64)(out))
		})
	})
}

var interleaveIndirect = archsimd.Uint64x4.InterleaveLoGrouped

func TestSIMDInterleaveIndirect(t *testing.T) {
	requireSIMD256(t)
	testInterleave[uint64](t, 4, 2, "InterleaveLoGrouped", func(x, y, out []uint64) {
		a, b := archsimd.LoadUint64x4Array((*[4]uint64)(x)), archsimd.LoadUint64x4Array((*[4]uint64)(y))
		interleaveIndirect(a, b).StoreArray((*[4]uint64)(out))
	})
}
