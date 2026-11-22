//go:build goexperiment.simd && amd64

package simd_test

import (
	"math"
	"simd/archsimd"
	"testing"
)

func TestSIMDPredicateEdges(t *testing.T) {
	requireSIMD128(t)
	xs := [4]float32{math.Float32frombits(0x7f800001), math.Float32frombits(0x80000000), math.Float32frombits(0xffc00001), float32(math.Inf(1))}
	if mask := archsimd.LoadFloat32x4Array(&xs).IsNaN().ToBits(); mask != 5 {
		t.Fatalf("NaN mask %#x, want 5", mask)
	}
	var bytes [16]byte
	if !archsimd.LoadUint8x16Array(&bytes).IsZero() {
		t.Fatal("zero vector")
	}
	for i := range bytes {
		bytes[i] = 0x80
		if archsimd.LoadUint8x16Array(&bytes).IsZero() {
			t.Fatalf("high bit in lane %d was lost", i)
		}
		bytes[i] = 0
	}
}
