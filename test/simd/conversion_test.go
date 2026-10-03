//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"math"
	"runtime"
	"simd/archsimd"
	"testing"
)

func TestSIMDConvertInt32(t *testing.T) {
	values := []float32{0, math.Float32frombits(1 << 31), 1.9, -1.9, 123456.75, -123456.75, 2147483520, -2147483648, 2147483648, -2147483904, float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))}
	for _, x := range values {
		var got [4]int32
		archsimd.BroadcastFloat32x4(x).ConvertToInt32().StoreArray(&got)
		var want int32
		switch {
		case math.IsNaN(float64(x)):
			if runtime.GOARCH == "amd64" {
				want = math.MinInt32
			}
		case x >= 2147483648:
			want = math.MaxInt32
			if runtime.GOARCH == "amd64" {
				want = math.MinInt32
			}
		case x < -2147483648:
			want = math.MinInt32
		default:
			want = int32(x)
		}
		for _, v := range got {
			if v != want {
				t.Fatalf("int32(%g): %d want %d", x, v, want)
			}
		}
	}
	for _, x := range []int32{math.MinInt32, math.MaxInt32, -16777217, 16777217, -1, 0, 1, 123456789} {
		var got [4]float32
		archsimd.BroadcastInt32x4(x).ConvertToFloat32().StoreArray(&got)
		for _, v := range got {
			if v != float32(x) {
				t.Fatalf("float32(%d): %g", x, v)
			}
		}
	}
}
