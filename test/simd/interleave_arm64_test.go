//go:build goexperiment.simd && arm64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDInterleave(t *testing.T) {
	t.Run("Int8x16/ConcatEven", func(t *testing.T) {
		testInterleave[int8](t, 16, 16, "ConcatEven", func(x, y, out []int8) {
			a, b := archsimd.LoadInt8x16Array((*[16]int8)(x)), archsimd.LoadInt8x16Array((*[16]int8)(y))
			a.ConcatEven(b).StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int16x8/ConcatEven", func(t *testing.T) {
		testInterleave[int16](t, 8, 8, "ConcatEven", func(x, y, out []int16) {
			a, b := archsimd.LoadInt16x8Array((*[8]int16)(x)), archsimd.LoadInt16x8Array((*[8]int16)(y))
			a.ConcatEven(b).StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int32x4/ConcatEven", func(t *testing.T) {
		testInterleave[int32](t, 4, 4, "ConcatEven", func(x, y, out []int32) {
			a, b := archsimd.LoadInt32x4Array((*[4]int32)(x)), archsimd.LoadInt32x4Array((*[4]int32)(y))
			a.ConcatEven(b).StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Int64x2/ConcatEven", func(t *testing.T) {
		testInterleave[int64](t, 2, 2, "ConcatEven", func(x, y, out []int64) {
			a, b := archsimd.LoadInt64x2Array((*[2]int64)(x)), archsimd.LoadInt64x2Array((*[2]int64)(y))
			a.ConcatEven(b).StoreArray((*[2]int64)(out))
		})
	})
	t.Run("Uint8x16/ConcatEven", func(t *testing.T) {
		testInterleave[uint8](t, 16, 16, "ConcatEven", func(x, y, out []uint8) {
			a, b := archsimd.LoadUint8x16Array((*[16]uint8)(x)), archsimd.LoadUint8x16Array((*[16]uint8)(y))
			a.ConcatEven(b).StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint16x8/ConcatEven", func(t *testing.T) {
		testInterleave[uint16](t, 8, 8, "ConcatEven", func(x, y, out []uint16) {
			a, b := archsimd.LoadUint16x8Array((*[8]uint16)(x)), archsimd.LoadUint16x8Array((*[8]uint16)(y))
			a.ConcatEven(b).StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint32x4/ConcatEven", func(t *testing.T) {
		testInterleave[uint32](t, 4, 4, "ConcatEven", func(x, y, out []uint32) {
			a, b := archsimd.LoadUint32x4Array((*[4]uint32)(x)), archsimd.LoadUint32x4Array((*[4]uint32)(y))
			a.ConcatEven(b).StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Uint64x2/ConcatEven", func(t *testing.T) {
		testInterleave[uint64](t, 2, 2, "ConcatEven", func(x, y, out []uint64) {
			a, b := archsimd.LoadUint64x2Array((*[2]uint64)(x)), archsimd.LoadUint64x2Array((*[2]uint64)(y))
			a.ConcatEven(b).StoreArray((*[2]uint64)(out))
		})
	})
	t.Run("Int8x16/ConcatOdd", func(t *testing.T) {
		testInterleave[int8](t, 16, 16, "ConcatOdd", func(x, y, out []int8) {
			a, b := archsimd.LoadInt8x16Array((*[16]int8)(x)), archsimd.LoadInt8x16Array((*[16]int8)(y))
			a.ConcatOdd(b).StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int16x8/ConcatOdd", func(t *testing.T) {
		testInterleave[int16](t, 8, 8, "ConcatOdd", func(x, y, out []int16) {
			a, b := archsimd.LoadInt16x8Array((*[8]int16)(x)), archsimd.LoadInt16x8Array((*[8]int16)(y))
			a.ConcatOdd(b).StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int32x4/ConcatOdd", func(t *testing.T) {
		testInterleave[int32](t, 4, 4, "ConcatOdd", func(x, y, out []int32) {
			a, b := archsimd.LoadInt32x4Array((*[4]int32)(x)), archsimd.LoadInt32x4Array((*[4]int32)(y))
			a.ConcatOdd(b).StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Int64x2/ConcatOdd", func(t *testing.T) {
		testInterleave[int64](t, 2, 2, "ConcatOdd", func(x, y, out []int64) {
			a, b := archsimd.LoadInt64x2Array((*[2]int64)(x)), archsimd.LoadInt64x2Array((*[2]int64)(y))
			a.ConcatOdd(b).StoreArray((*[2]int64)(out))
		})
	})
	t.Run("Uint8x16/ConcatOdd", func(t *testing.T) {
		testInterleave[uint8](t, 16, 16, "ConcatOdd", func(x, y, out []uint8) {
			a, b := archsimd.LoadUint8x16Array((*[16]uint8)(x)), archsimd.LoadUint8x16Array((*[16]uint8)(y))
			a.ConcatOdd(b).StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint16x8/ConcatOdd", func(t *testing.T) {
		testInterleave[uint16](t, 8, 8, "ConcatOdd", func(x, y, out []uint16) {
			a, b := archsimd.LoadUint16x8Array((*[8]uint16)(x)), archsimd.LoadUint16x8Array((*[8]uint16)(y))
			a.ConcatOdd(b).StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint32x4/ConcatOdd", func(t *testing.T) {
		testInterleave[uint32](t, 4, 4, "ConcatOdd", func(x, y, out []uint32) {
			a, b := archsimd.LoadUint32x4Array((*[4]uint32)(x)), archsimd.LoadUint32x4Array((*[4]uint32)(y))
			a.ConcatOdd(b).StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Uint64x2/ConcatOdd", func(t *testing.T) {
		testInterleave[uint64](t, 2, 2, "ConcatOdd", func(x, y, out []uint64) {
			a, b := archsimd.LoadUint64x2Array((*[2]uint64)(x)), archsimd.LoadUint64x2Array((*[2]uint64)(y))
			a.ConcatOdd(b).StoreArray((*[2]uint64)(out))
		})
	})
	t.Run("Int8x16/InterleaveEven", func(t *testing.T) {
		testInterleave[int8](t, 16, 16, "InterleaveEven", func(x, y, out []int8) {
			a, b := archsimd.LoadInt8x16Array((*[16]int8)(x)), archsimd.LoadInt8x16Array((*[16]int8)(y))
			a.InterleaveEven(b).StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int16x8/InterleaveEven", func(t *testing.T) {
		testInterleave[int16](t, 8, 8, "InterleaveEven", func(x, y, out []int16) {
			a, b := archsimd.LoadInt16x8Array((*[8]int16)(x)), archsimd.LoadInt16x8Array((*[8]int16)(y))
			a.InterleaveEven(b).StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int32x4/InterleaveEven", func(t *testing.T) {
		testInterleave[int32](t, 4, 4, "InterleaveEven", func(x, y, out []int32) {
			a, b := archsimd.LoadInt32x4Array((*[4]int32)(x)), archsimd.LoadInt32x4Array((*[4]int32)(y))
			a.InterleaveEven(b).StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Int64x2/InterleaveEven", func(t *testing.T) {
		testInterleave[int64](t, 2, 2, "InterleaveEven", func(x, y, out []int64) {
			a, b := archsimd.LoadInt64x2Array((*[2]int64)(x)), archsimd.LoadInt64x2Array((*[2]int64)(y))
			a.InterleaveEven(b).StoreArray((*[2]int64)(out))
		})
	})
	t.Run("Uint8x16/InterleaveEven", func(t *testing.T) {
		testInterleave[uint8](t, 16, 16, "InterleaveEven", func(x, y, out []uint8) {
			a, b := archsimd.LoadUint8x16Array((*[16]uint8)(x)), archsimd.LoadUint8x16Array((*[16]uint8)(y))
			a.InterleaveEven(b).StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint16x8/InterleaveEven", func(t *testing.T) {
		testInterleave[uint16](t, 8, 8, "InterleaveEven", func(x, y, out []uint16) {
			a, b := archsimd.LoadUint16x8Array((*[8]uint16)(x)), archsimd.LoadUint16x8Array((*[8]uint16)(y))
			a.InterleaveEven(b).StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint32x4/InterleaveEven", func(t *testing.T) {
		testInterleave[uint32](t, 4, 4, "InterleaveEven", func(x, y, out []uint32) {
			a, b := archsimd.LoadUint32x4Array((*[4]uint32)(x)), archsimd.LoadUint32x4Array((*[4]uint32)(y))
			a.InterleaveEven(b).StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Uint64x2/InterleaveEven", func(t *testing.T) {
		testInterleave[uint64](t, 2, 2, "InterleaveEven", func(x, y, out []uint64) {
			a, b := archsimd.LoadUint64x2Array((*[2]uint64)(x)), archsimd.LoadUint64x2Array((*[2]uint64)(y))
			a.InterleaveEven(b).StoreArray((*[2]uint64)(out))
		})
	})
	t.Run("Int8x16/InterleaveHi", func(t *testing.T) {
		testInterleave[int8](t, 16, 16, "InterleaveHi", func(x, y, out []int8) {
			a, b := archsimd.LoadInt8x16Array((*[16]int8)(x)), archsimd.LoadInt8x16Array((*[16]int8)(y))
			a.InterleaveHi(b).StoreArray((*[16]int8)(out))
		})
	})
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
	t.Run("Uint8x16/InterleaveHi", func(t *testing.T) {
		testInterleave[uint8](t, 16, 16, "InterleaveHi", func(x, y, out []uint8) {
			a, b := archsimd.LoadUint8x16Array((*[16]uint8)(x)), archsimd.LoadUint8x16Array((*[16]uint8)(y))
			a.InterleaveHi(b).StoreArray((*[16]uint8)(out))
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
	t.Run("Int8x16/InterleaveLo", func(t *testing.T) {
		testInterleave[int8](t, 16, 16, "InterleaveLo", func(x, y, out []int8) {
			a, b := archsimd.LoadInt8x16Array((*[16]int8)(x)), archsimd.LoadInt8x16Array((*[16]int8)(y))
			a.InterleaveLo(b).StoreArray((*[16]int8)(out))
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
	t.Run("Uint8x16/InterleaveLo", func(t *testing.T) {
		testInterleave[uint8](t, 16, 16, "InterleaveLo", func(x, y, out []uint8) {
			a, b := archsimd.LoadUint8x16Array((*[16]uint8)(x)), archsimd.LoadUint8x16Array((*[16]uint8)(y))
			a.InterleaveLo(b).StoreArray((*[16]uint8)(out))
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
	t.Run("Int8x16/InterleaveOdd", func(t *testing.T) {
		testInterleave[int8](t, 16, 16, "InterleaveOdd", func(x, y, out []int8) {
			a, b := archsimd.LoadInt8x16Array((*[16]int8)(x)), archsimd.LoadInt8x16Array((*[16]int8)(y))
			a.InterleaveOdd(b).StoreArray((*[16]int8)(out))
		})
	})
	t.Run("Int16x8/InterleaveOdd", func(t *testing.T) {
		testInterleave[int16](t, 8, 8, "InterleaveOdd", func(x, y, out []int16) {
			a, b := archsimd.LoadInt16x8Array((*[8]int16)(x)), archsimd.LoadInt16x8Array((*[8]int16)(y))
			a.InterleaveOdd(b).StoreArray((*[8]int16)(out))
		})
	})
	t.Run("Int32x4/InterleaveOdd", func(t *testing.T) {
		testInterleave[int32](t, 4, 4, "InterleaveOdd", func(x, y, out []int32) {
			a, b := archsimd.LoadInt32x4Array((*[4]int32)(x)), archsimd.LoadInt32x4Array((*[4]int32)(y))
			a.InterleaveOdd(b).StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Int64x2/InterleaveOdd", func(t *testing.T) {
		testInterleave[int64](t, 2, 2, "InterleaveOdd", func(x, y, out []int64) {
			a, b := archsimd.LoadInt64x2Array((*[2]int64)(x)), archsimd.LoadInt64x2Array((*[2]int64)(y))
			a.InterleaveOdd(b).StoreArray((*[2]int64)(out))
		})
	})
	t.Run("Uint8x16/InterleaveOdd", func(t *testing.T) {
		testInterleave[uint8](t, 16, 16, "InterleaveOdd", func(x, y, out []uint8) {
			a, b := archsimd.LoadUint8x16Array((*[16]uint8)(x)), archsimd.LoadUint8x16Array((*[16]uint8)(y))
			a.InterleaveOdd(b).StoreArray((*[16]uint8)(out))
		})
	})
	t.Run("Uint16x8/InterleaveOdd", func(t *testing.T) {
		testInterleave[uint16](t, 8, 8, "InterleaveOdd", func(x, y, out []uint16) {
			a, b := archsimd.LoadUint16x8Array((*[8]uint16)(x)), archsimd.LoadUint16x8Array((*[8]uint16)(y))
			a.InterleaveOdd(b).StoreArray((*[8]uint16)(out))
		})
	})
	t.Run("Uint32x4/InterleaveOdd", func(t *testing.T) {
		testInterleave[uint32](t, 4, 4, "InterleaveOdd", func(x, y, out []uint32) {
			a, b := archsimd.LoadUint32x4Array((*[4]uint32)(x)), archsimd.LoadUint32x4Array((*[4]uint32)(y))
			a.InterleaveOdd(b).StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Uint64x2/InterleaveOdd", func(t *testing.T) {
		testInterleave[uint64](t, 2, 2, "InterleaveOdd", func(x, y, out []uint64) {
			a, b := archsimd.LoadUint64x2Array((*[2]uint64)(x)), archsimd.LoadUint64x2Array((*[2]uint64)(y))
			a.InterleaveOdd(b).StoreArray((*[2]uint64)(out))
		})
	})
}

var interleaveIndirect = archsimd.Uint64x2.InterleaveOdd

func TestSIMDInterleaveIndirect(t *testing.T) {
	testInterleave[uint64](t, 2, 2, "InterleaveOdd", func(x, y, out []uint64) {
		a, b := archsimd.LoadUint64x2Array((*[2]uint64)(x)), archsimd.LoadUint64x2Array((*[2]uint64)(y))
		interleaveIndirect(a, b).StoreArray((*[2]uint64)(out))
	})
}
