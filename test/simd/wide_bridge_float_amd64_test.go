//go:build goexperiment.simd && amd64

package simd_test

import (
	"math"
	"runtime"
	"simd/archsimd"
	"testing"
)

func TestSIMD256BridgeFloatMinMax(t *testing.T) {
	requireSIMD256(t)
	bits32 := []uint32{0, 1 << 31, 1, 1<<31 | 1, 0x3f800000, 0xbf800000, 0x7f800000, 0xff800000, 0x7fc00001, 0xffc00002, 0x7f800001}
	for _, xb := range bits32 {
		for _, yb := range bits32 {
			x, y := math.Float32frombits(xb), math.Float32frombits(yb)
			a, b := archsimd.BroadcastFloat32x8(x), archsimd.BroadcastFloat32x8(y)
			for _, largest := range []bool{false, true} {
				v := a.Min(b)
				if largest {
					v = a.Max(b)
				}
				var out [8]float32
				v.StoreArray(&out)
				// Keep the original bits so the reference does not quiet a
				// signaling NaN. AMD64 chooses y for equal/unordered lanes.
				want := yb
				if largest && x > y || !largest && x < y {
					want = xb
				}
				for i, got := range out {
					if math.Float32bits(got) != want {
						t.Fatalf("f32 max=%v x=%x y=%x lane=%d got=%x want=%x", largest, xb, yb, i, math.Float32bits(got), want)
					}
				}
			}
		}
	}
	bits64 := []uint64{0, 1 << 63, 1, 1<<63 | 1, 0x3ff0000000000000, 0xbff0000000000000, 0x7ff0000000000000, 0xfff0000000000000, 0x7ff8000000000001, 0xfff8000000000002, 0x7ff0000000000001}
	for _, xb := range bits64 {
		for _, yb := range bits64 {
			x, y := math.Float64frombits(xb), math.Float64frombits(yb)
			a, b := archsimd.BroadcastFloat64x4(x), archsimd.BroadcastFloat64x4(y)
			for _, largest := range []bool{false, true} {
				v := a.Min(b)
				if largest {
					v = a.Max(b)
				}
				var out [4]float64
				v.StoreArray(&out)
				want := simdMinMaxReference(x, y, largest)
				for i, got := range out {
					if math.Float64bits(got) != math.Float64bits(want) {
						t.Fatalf("f64 max=%v x=%x y=%x lane=%d got=%x want=%x", largest, xb, yb, i, math.Float64bits(got), math.Float64bits(want))
					}
				}
			}
		}
	}
}

func TestSIMD256BridgeConvertInt32(t *testing.T) {
	requireSIMD256(t)
	values := []float32{0, math.Float32frombits(1 << 31), 1.9, -1.9, 123456.75, -123456.75, 2147483520, -2147483648, 2147483648, -2147483904, float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))}
	for _, x := range values {
		var got [8]int32
		archsimd.BroadcastFloat32x8(x).ConvertToInt32().StoreArray(&got)
		want := int32(math.MinInt32)
		if !math.IsNaN(float64(x)) && x >= -2147483648 && x < 2147483648 {
			want = int32(x)
		}
		for _, v := range got {
			if v != want {
				t.Fatalf("int32(%g): %d want %d", x, v, want)
			}
		}
	}
	for _, x := range []int32{math.MinInt32, math.MaxInt32, -16777217, 16777217, -1, 0, 1, 123456789} {
		var got [8]float32
		archsimd.BroadcastInt32x8(x).ConvertToFloat32().StoreArray(&got)
		for _, v := range got {
			if v != float32(x) {
				t.Fatalf("float32(%d): %g", x, v)
			}
		}
	}
}

func TestSIMD256BridgeMulAdd(t *testing.T) {
	requireSIMD256(t)
	if runtime.Compiler != "llgo" && !archsimd.X86.FMA() {
		t.Skip("official Go MulAdd requires FMA")
	}
	// Nonuniform lanes also exercise a captured receiver through the wide
	// function-value ABI. These finite products are exact in float64.
	a := [8]float32{1.25, -2.5, 3.75, -4.5, 7.25, -11.5, 13.75, -17.5}
	b := [8]float32{2.5, 3.25, -4.5, 5.75, -6.25, 7.5, -8.75, 9.25}
	c := [8]float32{-1, 2, -3, 4, -5, 6, -7, 8}
	x := archsimd.LoadFloat32x8Array(&a)
	method := x.MulAdd
	x = archsimd.BroadcastFloat32x8(100)
	var out [8]float32
	method(archsimd.LoadFloat32x8Array(&b), archsimd.LoadFloat32x8Array(&c)).StoreArray(&out)
	for i, got := range out {
		want := float32(math.FMA(float64(a[i]), float64(b[i]), float64(c[i])))
		if got != want {
			t.Fatalf("captured FMA lane %d = %g, want %g", i, got, want)
		}
	}
	for _, tc := range []struct{ x, y, z, fused float64 }{
		{1 + 0x1p-52, 1 - 0x1p-52, -1, -0x1p-104},
		{math.MaxFloat64, 2, -math.MaxFloat64, math.MaxFloat64},
		{0x1p-1022, 0x1p-52, 0, math.SmallestNonzeroFloat64},
		{math.Copysign(0, -1), 1, math.Copysign(0, -1), math.Copysign(0, -1)},
		{0, math.Inf(1), 1, math.NaN()},
		{math.Inf(1), 1, math.Inf(-1), math.NaN()},
	} {
		got := archsimd.BroadcastFloat64x4(tc.x).MulAdd(archsimd.BroadcastFloat64x4(tc.y), archsimd.BroadcastFloat64x4(tc.z))
		want := tc.fused
		var lanes [4]float64
		got.StoreArray(&lanes)
		for _, v := range lanes {
			if math.Float64bits(v) != math.Float64bits(want) && !(math.IsNaN(v) && math.IsNaN(want)) {
				t.Fatalf("float64 MulAdd(%g,%g,%g) = %g, want %g", tc.x, tc.y, tc.z, v, want)
			}
		}
	}
	for _, tc := range []struct{ x, y, z, fused float32 }{
		{1 + 0x1p-23, 1 - 0x1p-23, -1, -0x1p-46},
		{math.MaxFloat32, 2, -math.MaxFloat32, math.MaxFloat32},
		{1 + 0x1p-23, 1.5, -math.SmallestNonzeroFloat32, 1.5 + 0x1p-23},
		{1 + 0x1p-23, 1.5, math.SmallestNonzeroFloat32, 1.5 + 0x1p-22},
		{-1 - 0x1p-23, 1.5, math.SmallestNonzeroFloat32, -1.5 - 0x1p-23},
		{-1 - 0x1p-23, 1.5, -math.SmallestNonzeroFloat32, -1.5 - 0x1p-22},
		{0x1p-126, 0x1p-23, 0, math.SmallestNonzeroFloat32},
		{float32(math.Copysign(0, -1)), 1, float32(math.Copysign(0, -1)), float32(math.Copysign(0, -1))},
	} {
		got := archsimd.BroadcastFloat32x8(tc.x).MulAdd(archsimd.BroadcastFloat32x8(tc.y), archsimd.BroadcastFloat32x8(tc.z))
		want := tc.fused
		var lanes [8]float32
		got.StoreArray(&lanes)
		for _, v := range lanes {
			if math.Float32bits(v) != math.Float32bits(want) {
				t.Fatalf("float32 MulAdd(%g,%g,%g) = %g, want %g", tc.x, tc.y, tc.z, v, want)
			}
		}
	}
}
