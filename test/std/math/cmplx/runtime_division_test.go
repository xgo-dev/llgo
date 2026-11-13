package cmplx_test

import (
	"math"
	"testing"
)

//go:noinline
func runtimeDivide128(a, b complex128) complex128 { return a / b }

//go:noinline
func runtimeDivide64(a, b complex64) complex64 { return a / b }

// The expected IEEE-754 bits were obtained with Go's runtime division, for
// both widths. Keep signed zero; NaN payloads are deliberately unspecified.
func TestRuntimeComplexDivision(t *testing.T) {
	inf, nan, negzero := math.Inf(1), math.NaN(), math.Copysign(0, -1)
	cases := []struct {
		name    string
		a, b    complex128
		want128 [2]uint64
		want64  [2]uint32
	}{
		{"finite", 1 + 2i, 3 + 4i, [2]uint64{0x3fdc28f5c28f5c29, 0x3fb47ae147ae147b}, [2]uint32{0x3ee147ae, 0x3da3d70a}},
		{"real-dominant", 1 + 2i, 4 + 3i, [2]uint64{0x3fd999999999999a, 0x3fc999999999999a}, [2]uint32{0x3ecccccd, 0x3e4ccccd}},
		{"zero", 0, 1 + 2i, [2]uint64{0x0, 0x0}, [2]uint32{0x0, 0x0}},
		{"negative-zero", complex(negzero, negzero), 1 + 2i, [2]uint64{0x8000000000000000, 0x0}, [2]uint32{0x80000000, 0x0}},
		{"zero-denominator", 1 + 2i, 0, [2]uint64{0x7ff0000000000000, 0x7ff0000000000000}, [2]uint32{0x7f800000, 0x7f800000}},
		{"negative-zero-denominator", 1 + 2i, complex(negzero, negzero), [2]uint64{0xfff0000000000000, 0xfff0000000000000}, [2]uint32{0xff800000, 0xff800000}},
		{"zero-over-zero", 0, 0, [2]uint64{0x7ff8000000000000, 0x7ff8000000000000}, [2]uint32{0x7fc00000, 0x7fc00000}},
		{"nan-numerator", complex(nan, 1), 2 + 3i, [2]uint64{0x7ff8000000000000, 0x7ff8000000000000}, [2]uint32{0x7fc00000, 0x7fc00000}},
		{"nan-denominator", 1 + 2i, complex(nan, 3), [2]uint64{0x7ff8000000000000, 0x7ff8000000000000}, [2]uint32{0x7fc00000, 0x7fc00000}},
		{"infinite-real-numerator", complex(inf, 1), 2 + 3i, [2]uint64{0x7ff0000000000000, 0xfff0000000000000}, [2]uint32{0x7f800000, 0xff800000}},
		{"infinite-imaginary-numerator", complex(1, -inf), 2 + 3i, [2]uint64{0xfff0000000000000, 0xfff0000000000000}, [2]uint32{0xff800000, 0xff800000}},
		{"infinite-both-numerator", complex(inf, -inf), 2 + 3i, [2]uint64{0x7ff8000000000000, 0xfff0000000000000}, [2]uint32{0x7fc00000, 0xff800000}},
		{"infinite-nan-numerator", complex(inf, nan), 2 + 3i, [2]uint64{0x7ff0000000000000, 0xfff0000000000000}, [2]uint32{0x7f800000, 0xff800000}},
		{"infinite-real-denominator", 1 + 2i, complex(inf, 3), [2]uint64{0x0, 0x0}, [2]uint32{0x0, 0x0}},
		{"infinite-imaginary-denominator", 1 + 2i, complex(3, -inf), [2]uint64{0x8000000000000000, 0x0}, [2]uint32{0x80000000, 0x0}},
		{"infinite-both-denominator", 1 + 2i, complex(inf, -inf), [2]uint64{0x8000000000000000, 0x0}, [2]uint32{0x80000000, 0x0}},
		{"infinite-nan-denominator", 1 + 2i, complex(inf, nan), [2]uint64{0x0, 0x0}, [2]uint32{0x0, 0x0}},
		{"infinite-over-infinite", complex(inf, inf), complex(inf, -inf), [2]uint64{0x7ff8000000000000, 0x7ff8000000000000}, [2]uint32{0x7fc00000, 0x7fc00000}},
		{"max-finite-denominator", 1 + 1i, complex(math.MaxFloat64, math.MaxFloat64), [2]uint64{0x0, 0x0}, [2]uint32{0x0, 0x0}},
		{"subnormal-denominator", 1 + 2i, complex(math.SmallestNonzeroFloat64, math.SmallestNonzeroFloat64), [2]uint64{0x7ff0000000000000, 0x7ff0000000000000}, [2]uint32{0x7f800000, 0x7f800000}},
		{"subnormal-numerator", complex(math.SmallestNonzeroFloat64, math.SmallestNonzeroFloat64), 1, [2]uint64{0x1, 0x1}, [2]uint32{0x0, 0x0}},
		{"float32-max-denominator", 1 + 1i, complex(math.MaxFloat32, math.MaxFloat32), [2]uint64{0x37f0000010000010, 0x0}, [2]uint32{0x200000, 0x0}},
		{"float32-subnormal-denominator", 1 + 2i, complex(math.SmallestNonzeroFloat32, math.SmallestNonzeroFloat32), [2]uint64{0x4948000000000000, 0x4930000000000000}, [2]uint32{0x7f800000, 0x7f800000}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r128 := runtimeDivide128(tc.a, tc.b)
			r64 := runtimeDivide64(complex64(tc.a), complex64(tc.b))
			for i, v := range []float64{real(r128), imag(r128)} {
				if got := math.Float64bits(v); got != tc.want128[i] && !(math.IsNaN(v) && tc.want128[i] == 0x7ff8000000000000) {
					t.Errorf("complex128 component %d: bits %#x, want %#x", i, got, tc.want128[i])
				}
			}
			for i, v := range []float32{real(r64), imag(r64)} {
				if got := math.Float32bits(v); got != tc.want64[i] && !(math.IsNaN(float64(v)) && tc.want64[i] == 0x7fc00000) {
					t.Errorf("complex64 component %d: bits %#x, want %#x", i, got, tc.want64[i])
				}
			}
		})
	}
}
