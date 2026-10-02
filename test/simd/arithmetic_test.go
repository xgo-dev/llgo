//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"math"
	"simd/archsimd"
	"testing"
)

func TestSIMDArithmetic(t *testing.T) {
	x := [4]int32{math.MinInt32, -17, 65537, math.MaxInt32}
	y := [4]int32{-1, 7, 65537, 2}
	a, b := archsimd.LoadInt32x4Array(&x), archsimd.LoadInt32x4Array(&y)
	var mul, neg, abs [4]int32
	a.Mul(b).StoreArray(&mul)
	a.Neg().StoreArray(&neg)
	a.Abs().StoreArray(&abs)
	for i := range x {
		wantAbs := x[i]
		if wantAbs < 0 {
			wantAbs = -wantAbs
		}
		if mul[i] != x[i]*y[i] || neg[i] != -x[i] || abs[i] != wantAbs {
			t.Fatalf("lane %d: mul=%d neg=%d abs=%d", i, mul[i], neg[i], abs[i])
		}
	}
	var andnot, not [4]int32
	a.AndNot(b).StoreArray(&andnot)
	a.Not().StoreArray(&not)
	for i := range x {
		if andnot[i] != x[i]&^y[i] || not[i] != ^x[i] {
			t.Fatalf("bitwise lane %d: %x %x", i, andnot[i], not[i])
		}
	}
	fx, fy := [4]float32{1.5, -7, 0, 100}, [4]float32{2, 4, 0, -0.5}
	fa, fb := archsimd.LoadFloat32x4Array(&fx), archsimd.LoadFloat32x4Array(&fy)
	var product, quotient [4]float32
	fa.Mul(fb).StoreArray(&product)
	fa.Div(fb).StoreArray(&quotient)
	for i := range fx {
		if product[i] != fx[i]*fy[i] {
			t.Fatalf("float multiply lane %d", i)
		}
		want := fx[i] / fy[i]
		if quotient[i] != want && !(math.IsNaN(float64(quotient[i])) && math.IsNaN(float64(want))) {
			t.Fatalf("float divide lane %d = %g want %g", i, quotient[i], want)
		}
	}
}

func TestSIMDFloatUnary(t *testing.T) {
	for _, values := range [][2]float64{
		{-2.5, 3.5}, {-0.25, math.Copysign(0, -1)},
		{math.Inf(1), math.Inf(-1)}, {math.NaN(), 4},
		{math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64},
	} {
		v := archsimd.LoadFloat64x2Array(&values)
		for _, tc := range []struct {
			name string
			got  archsimd.Float64x2
			want func(float64) float64
		}{
			{"neg", v.Neg(), func(x float64) float64 { return -x }},
			{"abs", v.Abs(), math.Abs}, {"sqrt", v.Sqrt(), math.Sqrt},
			{"ceil", v.Ceil(), math.Ceil}, {"floor", v.Floor(), math.Floor},
			{"trunc", v.Trunc(), math.Trunc}, {"round", v.Round(), math.RoundToEven},
		} {
			var got [2]float64
			tc.got.StoreArray(&got)
			for i, x := range values {
				want := tc.want(x)
				if math.IsNaN(got[i]) && math.IsNaN(want) {
					continue
				}
				if math.Float64bits(got[i]) != math.Float64bits(want) {
					t.Fatalf("%s(%g) = %g (%x), want %g (%x)", tc.name, x, got[i], math.Float64bits(got[i]), want, math.Float64bits(want))
				}
			}
		}
	}
}

func TestSIMDBitcast(t *testing.T) {
	bits := [4]uint32{0x80000000, 0x7fc12345, 0x3f800000, 0xff800000}
	v := archsimd.LoadUint32x4Array(&bits)
	var got [4]uint32
	v.BitsToFloat32().ToBits().StoreArray(&got)
	if got != bits {
		t.Fatalf("float bitcast: %x", got)
	}
	v.BitsToInt32().ToBits().StoreArray(&got)
	if got != bits {
		t.Fatalf("integer bitcast: %x", got)
	}
	v.ReshapeToUint8s().ReshapeToUint16s().ReshapeToUint64s().ReshapeToUint32s().StoreArray(&got)
	if got != bits {
		t.Fatalf("reshape roundtrip: %x", got)
	}
	var bytes [16]byte
	v.ReshapeToUint8s().StoreArray(&bytes)
	for i, b := range bytes {
		if b != byte(bits[i/4]>>(8*(i%4))) {
			t.Fatalf("reshape byte %d = %x", i, b)
		}
	}
}
