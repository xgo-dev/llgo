//go:build goexperiment.simd && amd64

package simd_test

import (
	"bytes"
	"fmt"
	"runtime"
	"simd/archsimd"
	"strings"
	"testing"
	"unsafe"
)

type pairNumber interface {
	interleaveInteger | float32 | float64
}

func pairBytes[T pairNumber](v []T) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(&v[0])), len(v)*int(unsafe.Sizeof(v[0])))
}

func pairInputs[T pairNumber](n int) (x, y, out []T) {
	x, y, out = make([]T, n), make([]T, n), make([]T, n)
	// Distinct groups, signs and lane bits. The 32/64-bit patterns include NaN
	// payloads, signalling NaNs, infinities and -0; compare bytes, not floats.
	for which, v := range [][]T{x, y} {
		for i := range v {
			switch unsafe.Sizeof(v[i]) {
			case 4:
				patterns := [...]uint32{0x80000000, 0x7fc12345, 0x7f800001, 0xff800000, 0, 0xdeadbeef, 0x12345678, 0x3f800000}
				*(*uint32)(unsafe.Pointer(&v[i])) = patterns[(i+which*3)%len(patterns)]
			case 8:
				patterns := [...]uint64{0x8000000000000000, 0x7ff8123456789abc, 0x7ff0000000000001, 0xfff0000000000000, 0, 0xdeadbeef12345678, 0x0123456789abcdef, 0x3ff0000000000000}
				*(*uint64)(unsafe.Pointer(&v[i])) = patterns[(i+which*4)%len(patterns)]
			default:
				raw := pairBytes(v)
				for j := range raw {
					raw[j] = byte(17 + j*29 + which*71)
				}
			}
		}
	}
	return
}

// Test public Go helpers with every valid combination of lane selectors.
// The scalar oracle selects directly from the input slices, independently of
// the helpers' packed-immediate encoding and the compiler's shuffle masks.
func testPairScalars[T pairNumber](t *testing.T, n, group int, apply func([]T, []T, []uint8, []T)) {
	t.Helper()
	x, y, out := pairInputs[T](n)
	want := make([]T, n)
	selectors := make([]uint8, group)
	count := 1
	for range selectors {
		count *= 2 * group
	}
	for combination := 0; combination < count; combination++ {
		k := combination
		for i := range selectors {
			selectors[i] = uint8(k % (2 * group))
			k /= 2 * group
		}
		apply(x, y, selectors, out)
		for base := 0; base < n; base += group {
			input := append(append([]T{}, x[base:base+group]...), y[base:base+group]...)
			for i, index := range selectors {
				want[base+i] = input[index]
			}
		}
		if !bytes.Equal(pairBytes(out), pairBytes(want)) {
			t.Fatalf("selectors=%v: got %x want %x", selectors, pairBytes(out), pairBytes(want))
		}
	}
}

func testPairHalves[T pairNumber](t *testing.T, n int, apply func([]T, []T, uint8, uint8, []T)) {
	t.Helper()
	x, y, out := pairInputs[T](n)
	source := append(append([]T{}, x...), y...)
	for lo := uint8(0); lo < 4; lo++ {
		for hi := uint8(0); hi < 4; hi++ {
			apply(x, y, lo, hi, out)
			want := append(append([]T{}, source[int(lo)*n/2:(int(lo)+1)*n/2]...), source[int(hi)*n/2:(int(hi)+1)*n/2]...)
			if !bytes.Equal(pairBytes(out), pairBytes(want)) {
				t.Fatalf("halves %d,%d: got %x want %x", lo, hi, pairBytes(out), pairBytes(want))
			}
		}
	}
	for _, bad := range []uint8{4, 255} {
		for _, high := range []bool{false, true} {
			func() {
				defer func() {
					r := recover()
					if r == nil {
						t.Error("invalid half selector did not panic")
					} else if runtime.Compiler == "llgo" && !strings.Contains(fmt.Sprint(r), "out-of-range immediate") {
						t.Errorf("wrong panic: %v", r)
					}
				}()
				lo, hi := bad, uint8(0)
				if high {
					lo, hi = 0, bad
				}
				apply(x, y, lo, hi, out)
			}()
		}
	}
}

func TestSIMDPairShuffleScalars(t *testing.T) {
	requireSIMD128(t)
	t.Run("Int32x4", func(t *testing.T) {

		testPairScalars[int32](t, 4, 4, func(x, y []int32, s []uint8, out []int32) {
			a, b := archsimd.LoadInt32x4Array((*[4]int32)(x)), archsimd.LoadInt32x4Array((*[4]int32)(y))
			a.ConcatPermuteScalars(s[0], s[1], s[2], s[3], b).StoreArray((*[4]int32)(out))
		})
	})
	t.Run("Uint32x4", func(t *testing.T) {

		testPairScalars[uint32](t, 4, 4, func(x, y []uint32, s []uint8, out []uint32) {
			a, b := archsimd.LoadUint32x4Array((*[4]uint32)(x)), archsimd.LoadUint32x4Array((*[4]uint32)(y))
			a.ConcatPermuteScalars(s[0], s[1], s[2], s[3], b).StoreArray((*[4]uint32)(out))
		})
	})
	t.Run("Float32x4", func(t *testing.T) {

		testPairScalars[float32](t, 4, 4, func(x, y []float32, s []uint8, out []float32) {
			a, b := archsimd.LoadFloat32x4Array((*[4]float32)(x)), archsimd.LoadFloat32x4Array((*[4]float32)(y))
			a.ConcatPermuteScalars(s[0], s[1], s[2], s[3], b).StoreArray((*[4]float32)(out))
		})
	})
	t.Run("Int64x2", func(t *testing.T) {

		testPairScalars[int64](t, 2, 2, func(x, y []int64, s []uint8, out []int64) {
			a, b := archsimd.LoadInt64x2Array((*[2]int64)(x)), archsimd.LoadInt64x2Array((*[2]int64)(y))
			a.ConcatPermuteScalars(s[0], s[1], b).StoreArray((*[2]int64)(out))
		})
	})
	t.Run("Uint64x2", func(t *testing.T) {

		testPairScalars[uint64](t, 2, 2, func(x, y []uint64, s []uint8, out []uint64) {
			a, b := archsimd.LoadUint64x2Array((*[2]uint64)(x)), archsimd.LoadUint64x2Array((*[2]uint64)(y))
			a.ConcatPermuteScalars(s[0], s[1], b).StoreArray((*[2]uint64)(out))
		})
	})
	t.Run("Float64x2", func(t *testing.T) {

		testPairScalars[float64](t, 2, 2, func(x, y []float64, s []uint8, out []float64) {
			a, b := archsimd.LoadFloat64x2Array((*[2]float64)(x)), archsimd.LoadFloat64x2Array((*[2]float64)(y))
			a.ConcatPermuteScalars(s[0], s[1], b).StoreArray((*[2]float64)(out))
		})
	})
	t.Run("Int32x8", func(t *testing.T) {
		requireSIMD256(t)
		testPairScalars[int32](t, 8, 4, func(x, y []int32, s []uint8, out []int32) {
			a, b := archsimd.LoadInt32x8Array((*[8]int32)(x)), archsimd.LoadInt32x8Array((*[8]int32)(y))
			a.ConcatPermuteScalarsGrouped(s[0], s[1], s[2], s[3], b).StoreArray((*[8]int32)(out))
		})
	})
	t.Run("Uint32x8", func(t *testing.T) {
		requireSIMD256(t)
		testPairScalars[uint32](t, 8, 4, func(x, y []uint32, s []uint8, out []uint32) {
			a, b := archsimd.LoadUint32x8Array((*[8]uint32)(x)), archsimd.LoadUint32x8Array((*[8]uint32)(y))
			a.ConcatPermuteScalarsGrouped(s[0], s[1], s[2], s[3], b).StoreArray((*[8]uint32)(out))
		})
	})
	t.Run("Float32x8", func(t *testing.T) {
		requireSIMD256(t)
		testPairScalars[float32](t, 8, 4, func(x, y []float32, s []uint8, out []float32) {
			a, b := archsimd.LoadFloat32x8Array((*[8]float32)(x)), archsimd.LoadFloat32x8Array((*[8]float32)(y))
			a.ConcatPermuteScalarsGrouped(s[0], s[1], s[2], s[3], b).StoreArray((*[8]float32)(out))
		})
	})
	t.Run("Int64x4", func(t *testing.T) {
		requireSIMD256(t)
		testPairScalars[int64](t, 4, 2, func(x, y []int64, s []uint8, out []int64) {
			a, b := archsimd.LoadInt64x4Array((*[4]int64)(x)), archsimd.LoadInt64x4Array((*[4]int64)(y))
			a.ConcatPermuteScalarsGrouped(s[0], s[1], b).StoreArray((*[4]int64)(out))
		})
	})
	t.Run("Uint64x4", func(t *testing.T) {
		requireSIMD256(t)
		testPairScalars[uint64](t, 4, 2, func(x, y []uint64, s []uint8, out []uint64) {
			a, b := archsimd.LoadUint64x4Array((*[4]uint64)(x)), archsimd.LoadUint64x4Array((*[4]uint64)(y))
			a.ConcatPermuteScalarsGrouped(s[0], s[1], b).StoreArray((*[4]uint64)(out))
		})
	})
	t.Run("Float64x4", func(t *testing.T) {
		requireSIMD256(t)
		testPairScalars[float64](t, 4, 2, func(x, y []float64, s []uint8, out []float64) {
			a, b := archsimd.LoadFloat64x4Array((*[4]float64)(x)), archsimd.LoadFloat64x4Array((*[4]float64)(y))
			a.ConcatPermuteScalarsGrouped(s[0], s[1], b).StoreArray((*[4]float64)(out))
		})
	})
}

func TestSIMDPairShuffleHalves(t *testing.T) {
	requireSIMD256(t)
	t.Run("Int8x32", func(t *testing.T) {
		testPairHalves[int8](t, 32, func(x, y []int8, lo, hi uint8, out []int8) {
			a, b := archsimd.LoadInt8x32Array((*[32]int8)(x)), archsimd.LoadInt8x32Array((*[32]int8)(y))
			a.ConcatPermute128Scalars(lo, hi, b).StoreArray((*[32]int8)(out))
		})
	})
	t.Run("Uint8x32", func(t *testing.T) {
		testPairHalves[uint8](t, 32, func(x, y []uint8, lo, hi uint8, out []uint8) {
			a, b := archsimd.LoadUint8x32Array((*[32]uint8)(x)), archsimd.LoadUint8x32Array((*[32]uint8)(y))
			a.ConcatPermute128Scalars(lo, hi, b).StoreArray((*[32]uint8)(out))
		})
	})
	t.Run("Int16x16", func(t *testing.T) {
		testPairHalves[int16](t, 16, func(x, y []int16, lo, hi uint8, out []int16) {
			a, b := archsimd.LoadInt16x16Array((*[16]int16)(x)), archsimd.LoadInt16x16Array((*[16]int16)(y))
			a.ConcatPermute128Scalars(lo, hi, b).StoreArray((*[16]int16)(out))
		})
	})
	t.Run("Uint16x16", func(t *testing.T) {
		testPairHalves[uint16](t, 16, func(x, y []uint16, lo, hi uint8, out []uint16) {
			a, b := archsimd.LoadUint16x16Array((*[16]uint16)(x)), archsimd.LoadUint16x16Array((*[16]uint16)(y))
			a.ConcatPermute128Scalars(lo, hi, b).StoreArray((*[16]uint16)(out))
		})
	})
	t.Run("Int32x8", func(t *testing.T) {
		testPairHalves[int32](t, 8, func(x, y []int32, lo, hi uint8, out []int32) {
			a, b := archsimd.LoadInt32x8Array((*[8]int32)(x)), archsimd.LoadInt32x8Array((*[8]int32)(y))
			a.ConcatPermute128Scalars(lo, hi, b).StoreArray((*[8]int32)(out))
		})
	})
	t.Run("Uint32x8", func(t *testing.T) {
		testPairHalves[uint32](t, 8, func(x, y []uint32, lo, hi uint8, out []uint32) {
			a, b := archsimd.LoadUint32x8Array((*[8]uint32)(x)), archsimd.LoadUint32x8Array((*[8]uint32)(y))
			a.ConcatPermute128Scalars(lo, hi, b).StoreArray((*[8]uint32)(out))
		})
	})
	t.Run("Float32x8", func(t *testing.T) {
		testPairHalves[float32](t, 8, func(x, y []float32, lo, hi uint8, out []float32) {
			a, b := archsimd.LoadFloat32x8Array((*[8]float32)(x)), archsimd.LoadFloat32x8Array((*[8]float32)(y))
			a.ConcatPermute128Scalars(lo, hi, b).StoreArray((*[8]float32)(out))
		})
	})
	t.Run("Int64x4", func(t *testing.T) {
		testPairHalves[int64](t, 4, func(x, y []int64, lo, hi uint8, out []int64) {
			a, b := archsimd.LoadInt64x4Array((*[4]int64)(x)), archsimd.LoadInt64x4Array((*[4]int64)(y))
			a.ConcatPermute128Scalars(lo, hi, b).StoreArray((*[4]int64)(out))
		})
	})
	t.Run("Uint64x4", func(t *testing.T) {
		testPairHalves[uint64](t, 4, func(x, y []uint64, lo, hi uint8, out []uint64) {
			a, b := archsimd.LoadUint64x4Array((*[4]uint64)(x)), archsimd.LoadUint64x4Array((*[4]uint64)(y))
			a.ConcatPermute128Scalars(lo, hi, b).StoreArray((*[4]uint64)(out))
		})
	})
	t.Run("Float64x4", func(t *testing.T) {
		testPairHalves[float64](t, 4, func(x, y []float64, lo, hi uint8, out []float64) {
			a, b := archsimd.LoadFloat64x4Array((*[4]float64)(x)), archsimd.LoadFloat64x4Array((*[4]float64)(y))
			a.ConcatPermute128Scalars(lo, hi, b).StoreArray((*[4]float64)(out))
		})
	})
}

//go:noinline
func pairHalfIndirect(f func(uint8, uint8, archsimd.Uint64x4) archsimd.Uint64x4, lo, hi uint8, y archsimd.Uint64x4) archsimd.Uint64x4 {
	return f(lo, hi, y)
}

//go:noinline
func pairHalfFMV(x, y archsimd.Uint64x4) archsimd.Uint64x4 {
	if archsimd.X86.AVX2() {
		return x.ConcatPermute128Scalars(3, 0, y)
	}
	return x.ConcatPermute128Scalars(0, 3, y)
}

func TestSIMDPairShuffleCalls(t *testing.T) {
	requireSIMD256(t)
	testPairHalves[uint64](t, 4, func(x, y []uint64, lo, hi uint8, out []uint64) {
		a, b := archsimd.LoadUint64x4Array((*[4]uint64)(x)), archsimd.LoadUint64x4Array((*[4]uint64)(y))
		pairHalfIndirect(a.ConcatPermute128Scalars, lo, hi, b).StoreArray((*[4]uint64)(out))
	})
	a, b := archsimd.LoadUint64x4Array(&[4]uint64{10, 11, 20, 21}), archsimd.LoadUint64x4Array(&[4]uint64{30, 31, 40, 41})
	var out [4]uint64
	pairHalfFMV(a, b).StoreArray(&out)
	want := [4]uint64{10, 11, 40, 41}
	if archsimd.X86.AVX2() {
		want = [4]uint64{40, 41, 10, 11}
	}
	if out != want {
		t.Fatalf("FMV: got %v want %v", out, want)
	}
	// Literal invalid selectors must also retain a recoverable panic at O0/O2.
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("constant invalid half did not panic")
			} else if runtime.Compiler == "llgo" && !strings.Contains(fmt.Sprint(r), "out-of-range immediate") {
				t.Errorf("wrong panic: %v", r)
			}
		}()
		a.ConcatPermute128Scalars(4, 0, b).StoreArray(&out)
	}()
}
