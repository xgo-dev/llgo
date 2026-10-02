//go:build goexperiment.simd && (amd64 || arm64)

package simd_test

import (
	"math"
	"simd/archsimd"
	"testing"
)

func TestSIMDDemoteFloat64(t *testing.T) {
	for _, pair := range [][2]float64{{1.25, -1.25}, {math.Copysign(0, -1), 0}, {math.MaxFloat64, math.SmallestNonzeroFloat64}, {math.NaN(), math.Inf(-1)}} {
		var got [4]float32
		archsimd.LoadFloat64x2Array(&pair).ConvertToFloat32().StoreArray(&got)
		for i := 0; i < 2; i++ {
			want := float32(pair[i])
			if math.IsNaN(float64(want)) && math.IsNaN(float64(got[i])) {
				continue
			}
			if math.Float32bits(got[i]) != math.Float32bits(want) {
				t.Fatalf("lane %d: %g want %g", i, got[i], want)
			}
		}
		if math.Float32bits(got[2]) != 0 || math.Float32bits(got[3]) != 0 {
			t.Fatalf("nonzero upper lanes: %v", got)
		}
	}
}
