//go:build llgo && goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"
	_ "unsafe"
)

//go:linkname packedInt32x4PermuteScalars simd/archsimd.Int32x4.permuteScalars
func packedInt32x4PermuteScalars(archsimd.Int32x4, uint8) archsimd.Int32x4

//go:linkname packedUint32x4PermuteScalars simd/archsimd.Uint32x4.permuteScalars
func packedUint32x4PermuteScalars(archsimd.Uint32x4, uint8) archsimd.Uint32x4

//go:linkname packedInt16x8PermuteScalarsHi simd/archsimd.Int16x8.permuteScalarsHi
func packedInt16x8PermuteScalarsHi(archsimd.Int16x8, uint8) archsimd.Int16x8

//go:linkname packedUint16x8PermuteScalarsHi simd/archsimd.Uint16x8.permuteScalarsHi
func packedUint16x8PermuteScalarsHi(archsimd.Uint16x8, uint8) archsimd.Uint16x8

//go:linkname packedInt16x8PermuteScalarsLo simd/archsimd.Int16x8.permuteScalarsLo
func packedInt16x8PermuteScalarsLo(archsimd.Int16x8, uint8) archsimd.Int16x8

//go:linkname packedUint16x8PermuteScalarsLo simd/archsimd.Uint16x8.permuteScalarsLo
func packedUint16x8PermuteScalarsLo(archsimd.Uint16x8, uint8) archsimd.Uint16x8

//go:linkname packedInt32x8PermuteScalarsGrouped simd/archsimd.Int32x8.permuteScalarsGrouped
func packedInt32x8PermuteScalarsGrouped(archsimd.Int32x8, uint8) archsimd.Int32x8

//go:linkname packedUint32x8PermuteScalarsGrouped simd/archsimd.Uint32x8.permuteScalarsGrouped
func packedUint32x8PermuteScalarsGrouped(archsimd.Uint32x8, uint8) archsimd.Uint32x8

//go:linkname packedInt16x16PermuteScalarsHiGrouped simd/archsimd.Int16x16.permuteScalarsHiGrouped
func packedInt16x16PermuteScalarsHiGrouped(archsimd.Int16x16, uint8) archsimd.Int16x16

//go:linkname packedUint16x16PermuteScalarsHiGrouped simd/archsimd.Uint16x16.permuteScalarsHiGrouped
func packedUint16x16PermuteScalarsHiGrouped(archsimd.Uint16x16, uint8) archsimd.Uint16x16

//go:linkname packedInt16x16PermuteScalarsLoGrouped simd/archsimd.Int16x16.permuteScalarsLoGrouped
func packedInt16x16PermuteScalarsLoGrouped(archsimd.Int16x16, uint8) archsimd.Int16x16

//go:linkname packedUint16x16PermuteScalarsLoGrouped simd/archsimd.Uint16x16.permuteScalarsLoGrouped
func packedUint16x16PermuteScalarsLoGrouped(archsimd.Uint16x16, uint8) archsimd.Uint16x16

//go:noinline
func scalarPermutePackedIndirect[V any](f func(V, uint8) V, x V, selector uint8) V {
	return f(x, selector)
}

func TestSIMDScalarPermutePacked(t *testing.T) {
	t.Run("Int32x4/PermuteScalars", func(t *testing.T) {
		testScalarPermute[int32](t, 4, 4, 0, func(x []int32, s [4]uint8, out []int32, indirect bool) {
			a := archsimd.LoadInt32x4Array((*[4]int32)(x))
			packed := s[0]&3 | (s[1]&3)<<2 | (s[2]&3)<<4 | s[3]<<6
			if indirect {
				scalarPermutePackedIndirect(packedInt32x4PermuteScalars, a, packed).StoreArray((*[4]int32)(out))
			} else {
				packedInt32x4PermuteScalars(a, packed).StoreArray((*[4]int32)(out))
			}
		})
	})
	t.Run("Uint32x4/PermuteScalars", func(t *testing.T) {
		testScalarPermute[uint32](t, 4, 4, 0, func(x []uint32, s [4]uint8, out []uint32, indirect bool) {
			a := archsimd.LoadUint32x4Array((*[4]uint32)(x))
			packed := s[0]&3 | (s[1]&3)<<2 | (s[2]&3)<<4 | s[3]<<6
			if indirect {
				scalarPermutePackedIndirect(packedUint32x4PermuteScalars, a, packed).StoreArray((*[4]uint32)(out))
			} else {
				packedUint32x4PermuteScalars(a, packed).StoreArray((*[4]uint32)(out))
			}
		})
	})
	t.Run("Int16x8/PermuteScalarsHi", func(t *testing.T) {
		testScalarPermute[int16](t, 8, 8, 4, func(x []int16, s [4]uint8, out []int16, indirect bool) {
			a := archsimd.LoadInt16x8Array((*[8]int16)(x))
			packed := s[0]&3 | (s[1]&3)<<2 | (s[2]&3)<<4 | s[3]<<6
			if indirect {
				scalarPermutePackedIndirect(packedInt16x8PermuteScalarsHi, a, packed).StoreArray((*[8]int16)(out))
			} else {
				packedInt16x8PermuteScalarsHi(a, packed).StoreArray((*[8]int16)(out))
			}
		})
	})
	t.Run("Uint16x8/PermuteScalarsHi", func(t *testing.T) {
		testScalarPermute[uint16](t, 8, 8, 4, func(x []uint16, s [4]uint8, out []uint16, indirect bool) {
			a := archsimd.LoadUint16x8Array((*[8]uint16)(x))
			packed := s[0]&3 | (s[1]&3)<<2 | (s[2]&3)<<4 | s[3]<<6
			if indirect {
				scalarPermutePackedIndirect(packedUint16x8PermuteScalarsHi, a, packed).StoreArray((*[8]uint16)(out))
			} else {
				packedUint16x8PermuteScalarsHi(a, packed).StoreArray((*[8]uint16)(out))
			}
		})
	})
	t.Run("Int16x8/PermuteScalarsLo", func(t *testing.T) {
		testScalarPermute[int16](t, 8, 8, 0, func(x []int16, s [4]uint8, out []int16, indirect bool) {
			a := archsimd.LoadInt16x8Array((*[8]int16)(x))
			packed := s[0]&3 | (s[1]&3)<<2 | (s[2]&3)<<4 | s[3]<<6
			if indirect {
				scalarPermutePackedIndirect(packedInt16x8PermuteScalarsLo, a, packed).StoreArray((*[8]int16)(out))
			} else {
				packedInt16x8PermuteScalarsLo(a, packed).StoreArray((*[8]int16)(out))
			}
		})
	})
	t.Run("Uint16x8/PermuteScalarsLo", func(t *testing.T) {
		testScalarPermute[uint16](t, 8, 8, 0, func(x []uint16, s [4]uint8, out []uint16, indirect bool) {
			a := archsimd.LoadUint16x8Array((*[8]uint16)(x))
			packed := s[0]&3 | (s[1]&3)<<2 | (s[2]&3)<<4 | s[3]<<6
			if indirect {
				scalarPermutePackedIndirect(packedUint16x8PermuteScalarsLo, a, packed).StoreArray((*[8]uint16)(out))
			} else {
				packedUint16x8PermuteScalarsLo(a, packed).StoreArray((*[8]uint16)(out))
			}
		})
	})
	t.Run("Int32x8/PermuteScalarsGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testScalarPermute[int32](t, 8, 4, 0, func(x []int32, s [4]uint8, out []int32, indirect bool) {
			a := archsimd.LoadInt32x8Array((*[8]int32)(x))
			packed := s[0]&3 | (s[1]&3)<<2 | (s[2]&3)<<4 | s[3]<<6
			if indirect {
				scalarPermutePackedIndirect(packedInt32x8PermuteScalarsGrouped, a, packed).StoreArray((*[8]int32)(out))
			} else {
				packedInt32x8PermuteScalarsGrouped(a, packed).StoreArray((*[8]int32)(out))
			}
		})
	})
	t.Run("Uint32x8/PermuteScalarsGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testScalarPermute[uint32](t, 8, 4, 0, func(x []uint32, s [4]uint8, out []uint32, indirect bool) {
			a := archsimd.LoadUint32x8Array((*[8]uint32)(x))
			packed := s[0]&3 | (s[1]&3)<<2 | (s[2]&3)<<4 | s[3]<<6
			if indirect {
				scalarPermutePackedIndirect(packedUint32x8PermuteScalarsGrouped, a, packed).StoreArray((*[8]uint32)(out))
			} else {
				packedUint32x8PermuteScalarsGrouped(a, packed).StoreArray((*[8]uint32)(out))
			}
		})
	})
	t.Run("Int16x16/PermuteScalarsHiGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testScalarPermute[int16](t, 16, 8, 4, func(x []int16, s [4]uint8, out []int16, indirect bool) {
			a := archsimd.LoadInt16x16Array((*[16]int16)(x))
			packed := s[0]&3 | (s[1]&3)<<2 | (s[2]&3)<<4 | s[3]<<6
			if indirect {
				scalarPermutePackedIndirect(packedInt16x16PermuteScalarsHiGrouped, a, packed).StoreArray((*[16]int16)(out))
			} else {
				packedInt16x16PermuteScalarsHiGrouped(a, packed).StoreArray((*[16]int16)(out))
			}
		})
	})
	t.Run("Uint16x16/PermuteScalarsHiGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testScalarPermute[uint16](t, 16, 8, 4, func(x []uint16, s [4]uint8, out []uint16, indirect bool) {
			a := archsimd.LoadUint16x16Array((*[16]uint16)(x))
			packed := s[0]&3 | (s[1]&3)<<2 | (s[2]&3)<<4 | s[3]<<6
			if indirect {
				scalarPermutePackedIndirect(packedUint16x16PermuteScalarsHiGrouped, a, packed).StoreArray((*[16]uint16)(out))
			} else {
				packedUint16x16PermuteScalarsHiGrouped(a, packed).StoreArray((*[16]uint16)(out))
			}
		})
	})
	t.Run("Int16x16/PermuteScalarsLoGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testScalarPermute[int16](t, 16, 8, 0, func(x []int16, s [4]uint8, out []int16, indirect bool) {
			a := archsimd.LoadInt16x16Array((*[16]int16)(x))
			packed := s[0]&3 | (s[1]&3)<<2 | (s[2]&3)<<4 | s[3]<<6
			if indirect {
				scalarPermutePackedIndirect(packedInt16x16PermuteScalarsLoGrouped, a, packed).StoreArray((*[16]int16)(out))
			} else {
				packedInt16x16PermuteScalarsLoGrouped(a, packed).StoreArray((*[16]int16)(out))
			}
		})
	})
	t.Run("Uint16x16/PermuteScalarsLoGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testScalarPermute[uint16](t, 16, 8, 0, func(x []uint16, s [4]uint8, out []uint16, indirect bool) {
			a := archsimd.LoadUint16x16Array((*[16]uint16)(x))
			packed := s[0]&3 | (s[1]&3)<<2 | (s[2]&3)<<4 | s[3]<<6
			if indirect {
				scalarPermutePackedIndirect(packedUint16x16PermuteScalarsLoGrouped, a, packed).StoreArray((*[16]uint16)(out))
			} else {
				packedUint16x16PermuteScalarsLoGrouped(a, packed).StoreArray((*[16]uint16)(out))
			}
		})
	})
}
