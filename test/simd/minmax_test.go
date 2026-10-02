//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"math"
	"runtime"
	"simd/archsimd"
	"testing"
)

func simdMinMaxReference(x, y float64, largest bool) float64 {
	if runtime.GOARCH == "amd64" {
		if largest && x > y || !largest && x < y {
			return x
		}
		return y
	}
	if math.IsNaN(x) || math.IsNaN(y) {
		return math.NaN()
	}
	if largest {
		return math.Max(x, y)
	}
	return math.Min(x, y)
}

func TestSIMDFloatMinMax(t *testing.T) {
	bits32 := []uint32{0, 1 << 31, 1, 1<<31 | 1, 0x3f800000, 0xbf800000, 0x7f800000, 0xff800000, 0x7fc00001, 0xffc00002, 0x7f800001}
	for _, xb := range bits32 {
		for _, yb := range bits32 {
			x, y := math.Float32frombits(xb), math.Float32frombits(yb)
			a, b := archsimd.BroadcastFloat32x4(x), archsimd.BroadcastFloat32x4(y)
			for _, largest := range []bool{false, true} {
				v := a.Min(b)
				if largest {
					v = a.Max(b)
				}
				var out [4]float32
				v.StoreArray(&out)
				for i, got := range out {
					want := float32(simdMinMaxReference(float64(x), float64(y), largest))
					if runtime.GOARCH == "amd64" {
						// Select the original bits to avoid quieting a signaling NaN in the reference conversion.
						wb := yb
						if largest && x > y || !largest && x < y {
							wb = xb
						}
						if math.Float32bits(got) != wb {
							t.Fatalf("f32 max=%v x=%x y=%x lane=%d got=%x want=%x", largest, xb, yb, i, math.Float32bits(got), wb)
						}
					} else if !(math.IsNaN(float64(got)) && math.IsNaN(float64(want))) && math.Float32bits(got) != math.Float32bits(want) {
						t.Fatalf("f32 max=%v x=%x y=%x lane=%d got=%x want=%x", largest, xb, yb, i, math.Float32bits(got), math.Float32bits(want))
					}
				}
			}
		}
	}
	bits64 := []uint64{0, 1 << 63, 1, 1<<63 | 1, 0x3ff0000000000000, 0xbff0000000000000, 0x7ff0000000000000, 0xfff0000000000000, 0x7ff8000000000001, 0xfff8000000000002, 0x7ff0000000000001}
	for _, xb := range bits64 {
		for _, yb := range bits64 {
			x, y := math.Float64frombits(xb), math.Float64frombits(yb)
			a, b := archsimd.BroadcastFloat64x2(x), archsimd.BroadcastFloat64x2(y)
			for _, largest := range []bool{false, true} {
				v := a.Min(b)
				if largest {
					v = a.Max(b)
				}
				var out [2]float64
				v.StoreArray(&out)
				want := simdMinMaxReference(x, y, largest)
				for i, got := range out {
					if runtime.GOARCH != "amd64" && math.IsNaN(got) && math.IsNaN(want) {
						continue
					}
					if math.Float64bits(got) != math.Float64bits(want) {
						t.Fatalf("f64 max=%v x=%x y=%x lane=%d got=%x want=%x", largest, xb, yb, i, math.Float64bits(got), math.Float64bits(want))
					}
				}
			}
		}
	}
}
