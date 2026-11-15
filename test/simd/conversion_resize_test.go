//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"math"
	"runtime"
	"testing"
	"unsafe"
)

type conversionNumber interface {
	resizeInteger | float32 | float64
}

func testSIMDConversionShape[S, D conversionNumber](t *testing.T, op string, n, m int, apply func([]S, []D)) {
	t.Helper()
	var source S
	var dest D
	_, f32 := any(source).(float32)
	_, f64 := any(source).(float64)
	_, d32 := any(dest).(float32)
	_, d64 := any(dest).(float64)
	floats := []float64{0, math.Copysign(0, -1), 0.5, -0.5, 1.5, -1.5, 2.5, -2.5, 1.9, -1.9, math.NaN(), math.Inf(1), math.Inf(-1), math.SmallestNonzeroFloat32, -math.SmallestNonzeroFloat32, math.SmallestNonzeroFloat64, math.MaxFloat32, math.MaxFloat64}
	for _, power := range []int{24, 31, 32, 53, 63, 64} {
		v := math.Ldexp(1, power)
		floats = append(floats, v, -v, math.Nextafter(v, 0), math.Nextafter(v, math.Inf(1)), math.Nextafter(-v, 0), math.Nextafter(-v, math.Inf(-1)))
	}
	integers := []uint64{0, 1, 16777215, 16777216, 16777217, 1<<31 - 1, 1 << 31, 1<<32 - 1, 1<<53 - 1, 1 << 53, 1<<53 + 1, 1<<63 - 1, 1 << 63, ^uint64(0)}
	x, out := make([]S, n), make([]D, m)
	for sample := 0; sample < len(floats); sample++ {
		for i := range x {
			if f32 || f64 {
				x[i] = S(floats[(sample+i)%len(floats)])
			} else {
				x[i] = S(integers[(sample+i)%len(integers)])
			}
		}
		for i := range out {
			out[i] = D(37)
		}
		apply(x, out)
		for i, got := range out {
			var want D
			if i < n {
				x := x[i]
				want = D(x)
				v := float64(x)
				switch op {
				case "Ceil":
					want = D(math.Ceil(v))
				case "Floor":
					want = D(math.Floor(v))
				case "Trunc":
					want = D(math.Trunc(v))
				case "Round":
					want = D(math.RoundToEven(v))
				default:
					if (f32 || f64) && !d32 && !d64 {
						bits := int(unsafe.Sizeof(dest) * 8)
						unsigned := false
						switch any(dest).(type) {
						case uint8, uint16, uint32, uint64:
							unsigned = true
						}
						if unsigned {
							upper := uint64(1)<<bits - 1
							if math.IsNaN(v) || v <= -1 {
								want = 0
								if runtime.GOARCH == "amd64" {
									want = D(upper)
								}
							} else if v >= math.Ldexp(1, bits) {
								want = D(upper)
							} else if v < 0 {
								want = 0
							}
						} else {
							lower := int64(-1) << (bits - 1)
							upper := ^lower
							if math.IsNaN(v) {
								want = 0
								if runtime.GOARCH == "amd64" {
									want = D(lower)
								}
							} else if v >= math.Ldexp(1, bits-1) {
								want = D(upper)
								if runtime.GOARCH == "amd64" {
									want = D(lower)
								}
							} else if v < float64(lower) {
								want = D(lower)
							}
						}
					}
				}
			}
			same := got == want
			if d32 || d64 {
				same = (got != got && want != want) || math.Float64bits(float64(got)) == math.Float64bits(float64(want))
			}
			if !same {
				t.Fatalf("sample %d lane %d input=%v: got %v want %v", sample, i, x, got, want)
			}
		}
	}
}
