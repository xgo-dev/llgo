//go:build goexperiment.simd && (amd64 || arm64)

package simd_test

import (
	"math"
	"testing"
)

type horizontalNumber interface {
	int16 | uint16 | int32 | uint32 | int64 | uint64 | float32 | float64
}

// Use scalar pairs as the oracle, including independent upper and lower groups.
// The callbacks exercise real loads, arithmetic and stores, not vector layout.
func testHorizontal[T horizontalNumber](t *testing.T, n, group int, apply func([]T, []T, []T)) {
	t.Helper()
	x, y, got := make([]T, n), make([]T, n), make([]T, n)
	var zero T
	_, f32 := any(zero).(float32)
	_, f64 := any(zero).(float64)
	floats := []float64{0, math.Copysign(0, -1), 1, -1, math.Inf(1), math.Inf(-1), math.NaN(), math.SmallestNonzeroFloat32, -math.SmallestNonzeroFloat32, math.MaxFloat32, -math.MaxFloat32, 16777216, -16777216, math.SmallestNonzeroFloat64, math.MaxFloat64, -math.MaxFloat64, 9007199254740992}
	integers := []uint64{0, 1, 2, 0x7fff, 0x8000, 0xffff, 0x7fffffff, 0x80000000, 0xffffffff, 0x7fffffffffffffff, 0x8000000000000000, 0xffffffffffffffff}
	state := uint64(0x123456789abcdef)
	for sample := 0; sample < 320; sample++ {
		for i := range x {
			if f32 || f64 {
				pair := i / 2
				a, b := (sample+pair)%len(floats), (sample/len(floats)+3*pair)%len(floats)
				if i%2 == 0 {
					x[i], y[i] = T(floats[a]), T(floats[b])
				} else {
					x[i], y[i] = T(floats[b]), T(floats[a])
				}
			} else {
				state ^= state << 13
				state ^= state >> 7
				state ^= state << 17
				x[i] = T(state)
				y[i] = T(state >> 11)
				if sample < 144 {
					pair := i / 2
					a, b := (sample+pair)%len(integers), (sample/len(integers)+3*pair)%len(integers)
					if i%2 == 0 {
						x[i], y[i] = T(integers[a]), T(integers[b])
					} else {
						x[i], y[i] = T(integers[b]), T(integers[a])
					}
				}
			}
		}
		// A finite, distinct ramp makes incorrect lane/group selection unambiguous.
		if sample == 0 {
			for i := range x {
				x[i] = T(i + 1)
				y[i] = T(30 + 3*i)
			}
		}
		apply(x, y, got)
		for base := 0; base < n; base += group {
			for input, src := range [][]T{x, y} {
				for pair := 0; pair < group/2; pair++ {
					index := base + input*(group/2) + pair
					want := src[base+2*pair] + src[base+2*pair+1]
					value := got[index]
					same := value == want
					if f32 || f64 {
						same = (value != value && want != want) || math.Float64bits(float64(value)) == math.Float64bits(float64(want))
					}
					if !same {
						t.Fatalf("sample %d lane %d: got %v want %v; x=%v y=%v", sample, index, value, want, x, y)
					}
				}
			}
		}
	}
}
