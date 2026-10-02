//go:build goexperiment.simd && (arm64 || wasm || (llgo && amd64))

package simd_test

import (
	"math"
	"runtime"
	"simd/archsimd"
	"testing"
)

// Official amd64 requires AVX512 for these conversions. LLGo's baseline-safe
// LLVM legalization is checked against a scalar reference even without AVX512.
func TestSIMDConvertUint32(t *testing.T) {
	for _, x := range []float32{0, -0.9, -1, -1.9, 1.9, 2147483648, 4294967040, 4294967296, float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))} {
		var got [4]uint32
		archsimd.BroadcastFloat32x4(x).ConvertToUint32().StoreArray(&got)
		var want uint32
		switch {
		case math.IsNaN(float64(x)) || x <= -1:
			if runtime.GOARCH == "amd64" {
				want = math.MaxUint32
			}
		case x >= 4294967296:
			want = math.MaxUint32
		case x > 0:
			want = uint32(x)
		}
		for _, v := range got {
			if v != want {
				t.Fatalf("uint32(%g): %d want %d", x, v, want)
			}
		}
	}
	for _, x := range []uint32{0, 1, 16777217, 2147483648, math.MaxUint32} {
		var got [4]float32
		archsimd.BroadcastUint32x4(x).ConvertToFloat32().StoreArray(&got)
		for _, v := range got {
			if v != float32(x) {
				t.Fatalf("float32(%d): %g", x, v)
			}
		}
	}
}
