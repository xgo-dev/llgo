//go:build goexperiment.simd && arm64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDHorizontalAddARM64(t *testing.T) {
	requireHorizontal128(t)
	t.Run("Int64x2", func(t *testing.T) {
		testHorizontal[int64](t, 2, 2, func(x, y, out []int64) {
			a, b := archsimd.LoadInt64x2Array((*[2]int64)(x)), archsimd.LoadInt64x2Array((*[2]int64)(y))
			a.ConcatAddPairs(b).StoreArray((*[2]int64)(out))
		})
	})
	t.Run("Uint64x2", func(t *testing.T) {
		testHorizontal[uint64](t, 2, 2, func(x, y, out []uint64) {
			a, b := archsimd.LoadUint64x2Array((*[2]uint64)(x)), archsimd.LoadUint64x2Array((*[2]uint64)(y))
			a.ConcatAddPairs(b).StoreArray((*[2]uint64)(out))
		})
	})
}

func requireHorizontal128(t *testing.T) { t.Helper() }
