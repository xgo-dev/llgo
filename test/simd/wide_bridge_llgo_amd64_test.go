//go:build goexperiment.simd && llgo && amd64

package simd_test

import (
	"math"
	"simd/archsimd"
	"testing"
)

// These official intrinsics require AVX512-family features. Test LLGo's
// baseline legalization against scalar references, including cpu.all=off.
func TestSIMD256BridgeInteger64(t *testing.T) {
	t.Run("int64", func(t *testing.T) {
		testSIMD256BridgeInteger(t, func(a, b []int64, out [][]int64) {
			x, y := archsimd.LoadInt64x4(a), archsimd.LoadInt64x4(b)
			x.Min(y).Store(out[0])
			x.Max(y).Store(out[1])
			x.ConvertToUint64().ConvertToInt64().Store(out[2])
		})
	})
	t.Run("uint64", func(t *testing.T) {
		testSIMD256BridgeInteger(t, func(a, b []uint64, out [][]uint64) {
			x, y := archsimd.LoadUint64x4(a), archsimd.LoadUint64x4(b)
			x.Min(y).Store(out[0])
			x.Max(y).Store(out[1])
			x.ConvertToInt64().ConvertToUint64().Store(out[2])
		})
	})
}

func TestSIMD256BridgeCarryless(t *testing.T) {
	values := []uint64{0, 1, 3, ^uint64(0), 0x0123456789abcdef, 0xfedcba9876543210}
	for bit := uint(0); bit < 64; bit++ {
		values = append(values, 1<<bit)
	}
	for sample := range values {
		a := [4]uint64{values[sample], values[(sample+1)%len(values)], values[(sample+7)%len(values)], values[(sample+13)%len(values)]}
		b := [4]uint64{values[(sample*3+2)%len(values)], values[(sample*5+3)%len(values)], values[(sample*7+4)%len(values)], values[(sample*11+5)%len(values)]}
		x, y := archsimd.LoadUint64x4Array(&a), archsimd.LoadUint64x4Array(&b)
		for _, tc := range []struct {
			call func(archsimd.Uint64x4) archsimd.Uint64x4
			x, y int
		}{
			{x.CarrylessMultiplyEven, 0, 0}, {x.CarrylessMultiplyOdd, 1, 1},
			{x.CarrylessMultiplyOddEven, 1, 0}, {x.CarrylessMultiplyEvenOdd, 0, 1},
		} {
			var got, want [4]uint64
			tc.call(y).StoreArray(&got)
			for base := 0; base < 4; base += 2 {
				want[base], want[base+1] = carrylessReference(a[base+tc.x], b[base+tc.y])
			}
			if got != want {
				t.Fatalf("carryless selectors %d/%d: %x, want %x", tc.x, tc.y, got, want)
			}
		}
	}
}

func TestSIMD256BridgeConvert64(t *testing.T) {
	values := []float64{0, -0.9, -1, -1.9, 1.9, math.Ldexp(1, 63), -math.Ldexp(1, 63), math.Nextafter(math.Ldexp(1, 63), 0), math.Nextafter(math.Ldexp(1, 64), 0), math.Ldexp(1, 64), math.NaN(), math.Inf(1), math.Inf(-1)}
	for sample := range values {
		var input [4]float64
		for i := range input {
			input[i] = values[(sample+i)%len(values)]
		}
		x := archsimd.LoadFloat64x4Array(&input)
		var signed [4]int64
		var unsigned [4]uint64
		x.ConvertToInt64().StoreArray(&signed)
		x.ConvertToUint64().StoreArray(&unsigned)
		for i, value := range input {
			wantS, wantU := int64(math.MinInt64), uint64(math.MaxUint64)
			if !math.IsNaN(value) && value >= -math.Ldexp(1, 63) && value < math.Ldexp(1, 63) {
				wantS = int64(value)
			}
			if !math.IsNaN(value) && value > -1 && value < math.Ldexp(1, 64) {
				wantU = 0
				if value > 0 {
					wantU = uint64(value)
				}
			}
			if signed[i] != wantS || unsigned[i] != wantU {
				t.Fatalf("convert64 lane %d (%g) = %d/%d, want %d/%d", i, value, signed[i], unsigned[i], wantS, wantU)
			}
		}
	}
	integers := []uint64{0, 1, 1<<53 | 1, 1 << 63, math.MaxUint64}
	for sample := range integers {
		var input [4]uint64
		for i := range input {
			input[i] = integers[(sample+i)%len(integers)]
		}
		x := archsimd.LoadUint64x4Array(&input)
		var signed, unsigned [4]float64
		x.ConvertToFloat64().StoreArray(&unsigned)
		x.ConvertToInt64().ConvertToFloat64().StoreArray(&signed)
		for i, value := range input {
			if unsigned[i] != float64(value) || signed[i] != float64(int64(value)) {
				t.Fatalf("integer64 to float lane %d = %g/%g", i, signed[i], unsigned[i])
			}
		}
	}
}

func TestSIMD256BridgeConvertUint32(t *testing.T) {
	values := []float32{0, -0.9, -1, -1.9, 1.9, 0x1p31, math.Nextafter32(0x1p32, 0), 0x1p32, float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))}
	for sample := range values {
		var input [8]float32
		for i := range input {
			input[i] = values[(sample+i)%len(values)]
		}
		var got [8]uint32
		archsimd.LoadFloat32x8Array(&input).ConvertToUint32().StoreArray(&got)
		for i, value := range input {
			want := uint32(math.MaxUint32)
			if !math.IsNaN(float64(value)) && value > -1 && value < 0x1p32 {
				want = 0
				if value > 0 {
					want = uint32(value)
				}
			}
			if got[i] != want {
				t.Fatalf("convertUint32 lane %d (%g) = %d, want %d", i, value, got[i], want)
			}
		}
	}
	input := [8]uint32{0, 1, 1<<24 - 1, 1<<24 + 1, 1 << 31, 1<<31 + 1, math.MaxUint32 - 1, math.MaxUint32}
	var got [8]float32
	archsimd.LoadUint32x8Array(&input).ConvertToFloat32().StoreArray(&got)
	for i, value := range input {
		if got[i] != float32(value) {
			t.Fatalf("uint32 to float lane %d = %g, want %g", i, got[i], float32(value))
		}
	}
}
