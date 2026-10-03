//go:build goexperiment.simd && (arm64 || (llgo && amd64))

package simd_test

import (
	"math"
	"runtime"
	"simd/archsimd"
	"testing"
)

func TestSIMDConvert64(t *testing.T) {
	for _, x := range []float64{0, -0.9, -1, -1.9, 1.9, math.Ldexp(1, 63), -math.Ldexp(1, 63), math.Nextafter(math.Ldexp(1, 63), 0), math.Nextafter(math.Ldexp(1, 64), 0), math.Ldexp(1, 64), math.NaN(), math.Inf(1), math.Inf(-1)} {
		a := archsimd.BroadcastFloat64x2(x)
		var gotS [2]int64
		var gotU [2]uint64
		a.ConvertToInt64().StoreArray(&gotS)
		a.ConvertToUint64().StoreArray(&gotU)
		var wantS int64
		var wantU uint64
		switch {
		case math.IsNaN(x):
			if runtime.GOARCH == "amd64" {
				wantS = math.MinInt64
			}
		case x >= math.Ldexp(1, 63):
			wantS = math.MaxInt64
			if runtime.GOARCH == "amd64" {
				wantS = math.MinInt64
			}
		case x < -math.Ldexp(1, 63):
			wantS = math.MinInt64
		default:
			wantS = int64(x)
		}
		switch {
		case math.IsNaN(x) || x <= -1:
			if runtime.GOARCH == "amd64" {
				wantU = math.MaxUint64
			}
		case x >= math.Ldexp(1, 64):
			wantU = math.MaxUint64
		case x > 0:
			wantU = uint64(x)
		}
		for i := range gotS {
			if gotS[i] != wantS || gotU[i] != wantU {
				t.Fatalf("convert64(%g): %d/%d want %d/%d", x, gotS[i], gotU[i], wantS, wantU)
			}
		}
	}
	for _, x := range []uint64{0, 1, 1<<53 | 1, 1 << 63, math.MaxUint64} {
		var got [2]float64
		archsimd.BroadcastUint64x2(x).ConvertToFloat64().StoreArray(&got)
		if got[0] != float64(x) || got[1] != float64(x) {
			t.Fatalf("uint64 to float: %v", got)
		}
		archsimd.BroadcastInt64x2(int64(x)).ConvertToFloat64().StoreArray(&got)
		if got[0] != float64(int64(x)) || got[1] != float64(int64(x)) {
			t.Fatalf("int64 to float: %v", got)
		}
	}
}
