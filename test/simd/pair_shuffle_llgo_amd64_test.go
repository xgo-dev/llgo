//go:build llgo && goexperiment.simd && amd64

package simd_test

import (
	"bytes"
	"simd/archsimd"
	"testing"
	_ "unsafe"
)

//go:linkname pairPackedInt32x4 simd/archsimd.Int32x4.concatSelectedConstant
func pairPackedInt32x4(archsimd.Int32x4, uint8, archsimd.Int32x4) archsimd.Int32x4

//go:linkname pairPackedUint32x4 simd/archsimd.Uint32x4.concatSelectedConstant
func pairPackedUint32x4(archsimd.Uint32x4, uint8, archsimd.Uint32x4) archsimd.Uint32x4

//go:linkname pairPackedFloat32x4 simd/archsimd.Float32x4.concatSelectedConstant
func pairPackedFloat32x4(archsimd.Float32x4, uint8, archsimd.Float32x4) archsimd.Float32x4

//go:linkname pairPackedInt64x2 simd/archsimd.Int64x2.concatSelectedConstant
func pairPackedInt64x2(archsimd.Int64x2, uint8, archsimd.Int64x2) archsimd.Int64x2

//go:linkname pairPackedUint64x2 simd/archsimd.Uint64x2.concatSelectedConstant
func pairPackedUint64x2(archsimd.Uint64x2, uint8, archsimd.Uint64x2) archsimd.Uint64x2

//go:linkname pairPackedFloat64x2 simd/archsimd.Float64x2.concatSelectedConstant
func pairPackedFloat64x2(archsimd.Float64x2, uint8, archsimd.Float64x2) archsimd.Float64x2

//go:linkname pairPackedInt32x8 simd/archsimd.Int32x8.concatSelectedConstantGrouped
func pairPackedInt32x8(archsimd.Int32x8, uint8, archsimd.Int32x8) archsimd.Int32x8

//go:linkname pairPackedUint32x8 simd/archsimd.Uint32x8.concatSelectedConstantGrouped
func pairPackedUint32x8(archsimd.Uint32x8, uint8, archsimd.Uint32x8) archsimd.Uint32x8

//go:linkname pairPackedFloat32x8 simd/archsimd.Float32x8.concatSelectedConstantGrouped
func pairPackedFloat32x8(archsimd.Float32x8, uint8, archsimd.Float32x8) archsimd.Float32x8

//go:linkname pairPackedInt64x4 simd/archsimd.Int64x4.concatSelectedConstantGrouped
func pairPackedInt64x4(archsimd.Int64x4, uint8, archsimd.Int64x4) archsimd.Int64x4

//go:linkname pairPackedUint64x4 simd/archsimd.Uint64x4.concatSelectedConstantGrouped
func pairPackedUint64x4(archsimd.Uint64x4, uint8, archsimd.Uint64x4) archsimd.Uint64x4

//go:linkname pairPackedFloat64x4 simd/archsimd.Float64x4.concatSelectedConstantGrouped
func pairPackedFloat64x4(archsimd.Float64x4, uint8, archsimd.Float64x4) archsimd.Float64x4

// Exercise all 256 packed selectors, including ignored upper bits. Each
// oracle group explicitly selects its lower half from x and upper half from y.
func testPairPacked[T pairNumber](t *testing.T, n, bits int, apply func([]T, []T, uint8, []T)) {
	t.Helper()
	x, y, out := pairInputs[T](n)
	want := make([]T, n)
	for selector := 0; selector < 256; selector++ {
		apply(x, y, uint8(selector), out)
		if bits == 32 {
			for group := 0; group < n/4; group++ {
				base := 4 * group
				want[base] = x[base+(selector&3)]
				want[base+1] = x[base+((selector>>2)&3)]
				want[base+2] = y[base+((selector>>4)&3)]
				want[base+3] = y[base+((selector>>6)&3)]
			}
		} else {
			want[0] = x[selector&1]
			want[1] = y[(selector>>1)&1]
			if n == 4 {
				want[2] = x[2+((selector>>2)&1)]
				want[3] = y[2+((selector>>3)&1)]
			}
		}
		if !bytes.Equal(pairBytes(out), pairBytes(want)) {
			t.Fatalf("selector=%02x: got %x want %x", selector, pairBytes(out), pairBytes(want))
		}
	}
}

func TestSIMDPairShufflePacked(t *testing.T) {
	t.Run("Int32x4", func(t *testing.T) {
		testPairPacked[int32](t, 4, 32, func(x, y []int32, selector uint8, out []int32) {
			a, b := archsimd.LoadInt32x4Array((*[4]int32)(x)), archsimd.LoadInt32x4Array((*[4]int32)(y))
			pairPackedInt32x4(a, selector, b).StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Uint32x4", func(t *testing.T) {
		testPairPacked[uint32](t, 4, 32, func(x, y []uint32, selector uint8, out []uint32) {
			a, b := archsimd.LoadUint32x4Array((*[4]uint32)(x)), archsimd.LoadUint32x4Array((*[4]uint32)(y))
			pairPackedUint32x4(a, selector, b).StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Float32x4", func(t *testing.T) {
		testPairPacked[float32](t, 4, 32, func(x, y []float32, selector uint8, out []float32) {
			a, b := archsimd.LoadFloat32x4Array((*[4]float32)(x)), archsimd.LoadFloat32x4Array((*[4]float32)(y))
			pairPackedFloat32x4(a, selector, b).StoreArray((*[4]float32)(out))
		})
	})
	t.Run("Int64x2", func(t *testing.T) {
		testPairPacked[int64](t, 2, 64, func(x, y []int64, selector uint8, out []int64) {
			a, b := archsimd.LoadInt64x2Array((*[2]int64)(x)), archsimd.LoadInt64x2Array((*[2]int64)(y))
			pairPackedInt64x2(a, selector, b).StoreArray((*[2]int64)(out))
		})
	})
	t.Run("Uint64x2", func(t *testing.T) {
		testPairPacked[uint64](t, 2, 64, func(x, y []uint64, selector uint8, out []uint64) {
			a, b := archsimd.LoadUint64x2Array((*[2]uint64)(x)), archsimd.LoadUint64x2Array((*[2]uint64)(y))
			pairPackedUint64x2(a, selector, b).StoreArray((*[2]uint64)(out))
		})
	})
	t.Run("Float64x2", func(t *testing.T) {
		testPairPacked[float64](t, 2, 64, func(x, y []float64, selector uint8, out []float64) {
			a, b := archsimd.LoadFloat64x2Array((*[2]float64)(x)), archsimd.LoadFloat64x2Array((*[2]float64)(y))
			pairPackedFloat64x2(a, selector, b).StoreArray((*[2]float64)(out))
		})
	})
	t.Run("Int32x8", func(t *testing.T) {
		requireSIMD256(t)
		testPairPacked[int32](t, 8, 32, func(x, y []int32, selector uint8, out []int32) {
			a, b := archsimd.LoadInt32x8Array((*[8]int32)(x)), archsimd.LoadInt32x8Array((*[8]int32)(y))
			pairPackedInt32x8(a, selector, b).StoreArray((*[8]int32)(out))
		})
	})
	t.Run("Uint32x8", func(t *testing.T) {
		requireSIMD256(t)
		testPairPacked[uint32](t, 8, 32, func(x, y []uint32, selector uint8, out []uint32) {
			a, b := archsimd.LoadUint32x8Array((*[8]uint32)(x)), archsimd.LoadUint32x8Array((*[8]uint32)(y))
			pairPackedUint32x8(a, selector, b).StoreArray((*[8]uint32)(out))
		})
	})
	t.Run("Float32x8", func(t *testing.T) {
		requireSIMD256(t)
		testPairPacked[float32](t, 8, 32, func(x, y []float32, selector uint8, out []float32) {
			a, b := archsimd.LoadFloat32x8Array((*[8]float32)(x)), archsimd.LoadFloat32x8Array((*[8]float32)(y))
			pairPackedFloat32x8(a, selector, b).StoreArray((*[8]float32)(out))
		})
	})
	t.Run("Int64x4", func(t *testing.T) {
		requireSIMD256(t)
		testPairPacked[int64](t, 4, 64, func(x, y []int64, selector uint8, out []int64) {
			a, b := archsimd.LoadInt64x4Array((*[4]int64)(x)), archsimd.LoadInt64x4Array((*[4]int64)(y))
			pairPackedInt64x4(a, selector, b).StoreArray((*[4]int64)(out))
		})
	})
	t.Run("Uint64x4", func(t *testing.T) {
		requireSIMD256(t)
		testPairPacked[uint64](t, 4, 64, func(x, y []uint64, selector uint8, out []uint64) {
			a, b := archsimd.LoadUint64x4Array((*[4]uint64)(x)), archsimd.LoadUint64x4Array((*[4]uint64)(y))
			pairPackedUint64x4(a, selector, b).StoreArray((*[4]uint64)(out))
		})
	})
	t.Run("Float64x4", func(t *testing.T) {
		requireSIMD256(t)
		testPairPacked[float64](t, 4, 64, func(x, y []float64, selector uint8, out []float64) {
			a, b := archsimd.LoadFloat64x4Array((*[4]float64)(x)), archsimd.LoadFloat64x4Array((*[4]float64)(y))
			pairPackedFloat64x4(a, selector, b).StoreArray((*[4]float64)(out))
		})
	})
}
