//go:build goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

// Scalar oracle selects only four lanes in each 128-bit group. The other
// four word lanes are sentinels, and groups contain distinct bit patterns.
func testScalarPermute[T interleaveInteger](t *testing.T, n, group, start int, apply func([]T, [4]uint8, []T, bool)) {
	t.Helper()
	x, out, want := make([]T, n), make([]T, n), make([]T, n)
	for i := range x {
		x[i] = T(uint32(0x80018001) + uint32(i)*0x10231023)
	}
	for _, indirect := range []bool{false, true} {
		for packed := 0; packed < 256; packed++ {
			s := [4]uint8{uint8(packed & 3), uint8((packed >> 2) & 3), uint8((packed >> 4) & 3), uint8(packed >> 6)}
			for _, upperBits := range []bool{false, true} {
				if upperBits {
					s[0] |= 0xfc
					s[1] |= 0x80
					s[2] |= 0x44
					s[3] |= 0x10
				}
				copy(want, x)
				for base := 0; base < n; base += group {
					for j := 0; j < 4; j++ {
						want[base+start+j] = x[base+start+int(s[j]&3)]
					}
				}
				apply(x, s, out, indirect)
				for i := range out {
					if out[i] != want[i] {
						t.Fatalf("selectors=%v indirect=%v lane=%d: got %v want %v", s, indirect, i, out, want)
					}
				}
			}
		}
	}
}

//go:noinline
func scalarPermuteIndirect[V any](f func(uint8, uint8, uint8, uint8) V, s [4]uint8) V {
	return f(s[0], s[1], s[2], s[3])
}

func TestSIMDScalarPermute(t *testing.T) {
	t.Run("Int32x4/PermuteScalars", func(t *testing.T) {
		requireSIMD128(t)
		testScalarPermute[int32](t, 4, 4, 0, func(x []int32, s [4]uint8, out []int32, indirect bool) {
			a := archsimd.LoadInt32x4Array((*[4]int32)(x))
			if indirect {
				scalarPermuteIndirect(a.PermuteScalars, s).StoreArray((*[4]int32)(out))
			} else {
				a.PermuteScalars(s[0], s[1], s[2], s[3]).StoreArray((*[4]int32)(out))
			}
		})
	})
	t.Run("Uint32x4/PermuteScalars", func(t *testing.T) {
		requireSIMD128(t)
		testScalarPermute[uint32](t, 4, 4, 0, func(x []uint32, s [4]uint8, out []uint32, indirect bool) {
			a := archsimd.LoadUint32x4Array((*[4]uint32)(x))
			if indirect {
				scalarPermuteIndirect(a.PermuteScalars, s).StoreArray((*[4]uint32)(out))
			} else {
				a.PermuteScalars(s[0], s[1], s[2], s[3]).StoreArray((*[4]uint32)(out))
			}
		})
	})
	t.Run("Int16x8/PermuteScalarsHi", func(t *testing.T) {
		requireSIMD128(t)
		testScalarPermute[int16](t, 8, 8, 4, func(x []int16, s [4]uint8, out []int16, indirect bool) {
			a := archsimd.LoadInt16x8Array((*[8]int16)(x))
			if indirect {
				scalarPermuteIndirect(a.PermuteScalarsHi, s).StoreArray((*[8]int16)(out))
			} else {
				a.PermuteScalarsHi(s[0], s[1], s[2], s[3]).StoreArray((*[8]int16)(out))
			}
		})
	})
	t.Run("Uint16x8/PermuteScalarsHi", func(t *testing.T) {
		requireSIMD128(t)
		testScalarPermute[uint16](t, 8, 8, 4, func(x []uint16, s [4]uint8, out []uint16, indirect bool) {
			a := archsimd.LoadUint16x8Array((*[8]uint16)(x))
			if indirect {
				scalarPermuteIndirect(a.PermuteScalarsHi, s).StoreArray((*[8]uint16)(out))
			} else {
				a.PermuteScalarsHi(s[0], s[1], s[2], s[3]).StoreArray((*[8]uint16)(out))
			}
		})
	})
	t.Run("Int16x8/PermuteScalarsLo", func(t *testing.T) {
		requireSIMD128(t)
		testScalarPermute[int16](t, 8, 8, 0, func(x []int16, s [4]uint8, out []int16, indirect bool) {
			a := archsimd.LoadInt16x8Array((*[8]int16)(x))
			if indirect {
				scalarPermuteIndirect(a.PermuteScalarsLo, s).StoreArray((*[8]int16)(out))
			} else {
				a.PermuteScalarsLo(s[0], s[1], s[2], s[3]).StoreArray((*[8]int16)(out))
			}
		})
	})
	t.Run("Uint16x8/PermuteScalarsLo", func(t *testing.T) {
		requireSIMD128(t)
		testScalarPermute[uint16](t, 8, 8, 0, func(x []uint16, s [4]uint8, out []uint16, indirect bool) {
			a := archsimd.LoadUint16x8Array((*[8]uint16)(x))
			if indirect {
				scalarPermuteIndirect(a.PermuteScalarsLo, s).StoreArray((*[8]uint16)(out))
			} else {
				a.PermuteScalarsLo(s[0], s[1], s[2], s[3]).StoreArray((*[8]uint16)(out))
			}
		})
	})
	t.Run("Int32x8/PermuteScalarsGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testScalarPermute[int32](t, 8, 4, 0, func(x []int32, s [4]uint8, out []int32, indirect bool) {
			a := archsimd.LoadInt32x8Array((*[8]int32)(x))
			if indirect {
				scalarPermuteIndirect(a.PermuteScalarsGrouped, s).StoreArray((*[8]int32)(out))
			} else {
				a.PermuteScalarsGrouped(s[0], s[1], s[2], s[3]).StoreArray((*[8]int32)(out))
			}
		})
	})
	t.Run("Uint32x8/PermuteScalarsGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testScalarPermute[uint32](t, 8, 4, 0, func(x []uint32, s [4]uint8, out []uint32, indirect bool) {
			a := archsimd.LoadUint32x8Array((*[8]uint32)(x))
			if indirect {
				scalarPermuteIndirect(a.PermuteScalarsGrouped, s).StoreArray((*[8]uint32)(out))
			} else {
				a.PermuteScalarsGrouped(s[0], s[1], s[2], s[3]).StoreArray((*[8]uint32)(out))
			}
		})
	})
	t.Run("Int16x16/PermuteScalarsHiGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testScalarPermute[int16](t, 16, 8, 4, func(x []int16, s [4]uint8, out []int16, indirect bool) {
			a := archsimd.LoadInt16x16Array((*[16]int16)(x))
			if indirect {
				scalarPermuteIndirect(a.PermuteScalarsHiGrouped, s).StoreArray((*[16]int16)(out))
			} else {
				a.PermuteScalarsHiGrouped(s[0], s[1], s[2], s[3]).StoreArray((*[16]int16)(out))
			}
		})
	})
	t.Run("Uint16x16/PermuteScalarsHiGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testScalarPermute[uint16](t, 16, 8, 4, func(x []uint16, s [4]uint8, out []uint16, indirect bool) {
			a := archsimd.LoadUint16x16Array((*[16]uint16)(x))
			if indirect {
				scalarPermuteIndirect(a.PermuteScalarsHiGrouped, s).StoreArray((*[16]uint16)(out))
			} else {
				a.PermuteScalarsHiGrouped(s[0], s[1], s[2], s[3]).StoreArray((*[16]uint16)(out))
			}
		})
	})
	t.Run("Int16x16/PermuteScalarsLoGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testScalarPermute[int16](t, 16, 8, 0, func(x []int16, s [4]uint8, out []int16, indirect bool) {
			a := archsimd.LoadInt16x16Array((*[16]int16)(x))
			if indirect {
				scalarPermuteIndirect(a.PermuteScalarsLoGrouped, s).StoreArray((*[16]int16)(out))
			} else {
				a.PermuteScalarsLoGrouped(s[0], s[1], s[2], s[3]).StoreArray((*[16]int16)(out))
			}
		})
	})
	t.Run("Uint16x16/PermuteScalarsLoGrouped", func(t *testing.T) {
		requireSIMD256(t)
		testScalarPermute[uint16](t, 16, 8, 0, func(x []uint16, s [4]uint8, out []uint16, indirect bool) {
			a := archsimd.LoadUint16x16Array((*[16]uint16)(x))
			if indirect {
				scalarPermuteIndirect(a.PermuteScalarsLoGrouped, s).StoreArray((*[16]uint16)(out))
			} else {
				a.PermuteScalarsLoGrouped(s[0], s[1], s[2], s[3]).StoreArray((*[16]uint16)(out))
			}
		})
	})
}

//go:noinline
func scalarPermuteFMVDwords(x archsimd.Int32x8, a, b, c, d uint8) archsimd.Int32x8 {
	if archsimd.X86.AVX2() {
		return x.PermuteScalarsGrouped(a, b, c, d)
	}
	return x.PermuteScalarsGrouped(a, b, c, d)
}

//go:noinline
func scalarPermuteFMVWordsHi(x archsimd.Uint16x16, a, b, c, d uint8) archsimd.Uint16x16 {
	if archsimd.X86.AVX2() {
		return x.PermuteScalarsHiGrouped(a, b, c, d)
	}
	return x.PermuteScalarsHiGrouped(a, b, c, d)
}

//go:noinline
func scalarPermuteFMVWordsLo(x archsimd.Uint16x16, a, b, c, d uint8) archsimd.Uint16x16 {
	if archsimd.X86.AVX2() {
		return x.PermuteScalarsLoGrouped(a, b, c, d)
	}
	return x.PermuteScalarsLoGrouped(a, b, c, d)
}

func TestSIMDScalarPermuteFMV(t *testing.T) {
	requireSIMD256(t)
	t.Run("Int32x8/PermuteScalarsGrouped", func(t *testing.T) {
		testScalarPermute[int32](t, 8, 4, 0, func(x []int32, s [4]uint8, out []int32, _ bool) {
			a := archsimd.LoadInt32x8Array((*[8]int32)(x))
			scalarPermuteFMVDwords(a, s[0], s[1], s[2], s[3]).StoreArray((*[8]int32)(out))
		})
	})
	t.Run("Uint16x16/PermuteScalarsHiGrouped", func(t *testing.T) {
		testScalarPermute[uint16](t, 16, 8, 4, func(x []uint16, s [4]uint8, out []uint16, _ bool) {
			a := archsimd.LoadUint16x16Array((*[16]uint16)(x))
			scalarPermuteFMVWordsHi(a, s[0], s[1], s[2], s[3]).StoreArray((*[16]uint16)(out))
		})
	})
	t.Run("Uint16x16/PermuteScalarsLoGrouped", func(t *testing.T) {
		testScalarPermute[uint16](t, 16, 8, 0, func(x []uint16, s [4]uint8, out []uint16, _ bool) {
			a := archsimd.LoadUint16x16Array((*[16]uint16)(x))
			scalarPermuteFMVWordsLo(a, s[0], s[1], s[2], s[3]).StoreArray((*[16]uint16)(out))
		})
	})
}
