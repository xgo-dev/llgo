package math_test

import (
	"math"
	"testing"
)

const tolerance = 1e-12

func assertFloatNear(t *testing.T, got, want, tol float64) {
	t.Helper()
	if math.IsNaN(got) && math.IsNaN(want) {
		return
	}
	if math.IsInf(got, 0) && math.IsInf(want, 0) && math.Signbit(got) == math.Signbit(want) {
		return
	}
	diff := math.Abs(got - want)
	if diff > tol {
		t.Errorf("got %v, want %v (diff %v > tolerance %v)", got, want, diff, tol)
	}
}

func TestUnaryFloatFunctions(t *testing.T) {
	tests := []struct {
		name  string
		fn    func(float64) float64
		input float64
		want  float64
		tol   float64
	}{
		{name: "Abs", fn: math.Abs, input: -3.5, want: 3.5, tol: 0},
		{name: "Acos", fn: math.Acos, input: 0.5, want: math.Pi / 3, tol: 1e-12},
		{name: "Acosh", fn: math.Acosh, input: 2, want: 1.3169578969248166, tol: 1e-12},
		{name: "Asin", fn: math.Asin, input: 0.5, want: math.Pi / 6, tol: 1e-12},
		{name: "Asinh", fn: math.Asinh, input: 1, want: 0.881373587019543, tol: 1e-12},
		{name: "Atan", fn: math.Atan, input: 1, want: math.Pi / 4, tol: 1e-12},
		{name: "Atanh", fn: math.Atanh, input: 0.5, want: 0.5493061443340549, tol: 1e-12},
		{name: "Cbrt", fn: math.Cbrt, input: 27, want: 3, tol: 1e-12},
		{name: "Ceil", fn: math.Ceil, input: -1.2, want: -1, tol: 0},
		{name: "Cos", fn: math.Cos, input: 0, want: 1, tol: 0},
		{name: "Cosh", fn: math.Cosh, input: 1, want: (math.Exp(1) + math.Exp(-1)) / 2, tol: 1e-12},
		{name: "Erf", fn: math.Erf, input: 1, want: 0.8427007929497149, tol: 1e-12},
		{name: "Erfc", fn: math.Erfc, input: 1, want: 0.1572992070502851, tol: 1e-12},
		{name: "Erfcinv", fn: math.Erfcinv, input: 0.5, want: 0.4769362762044699, tol: 1e-12},
		{name: "Erfinv", fn: math.Erfinv, input: 0.5, want: 0.4769362762044699, tol: 1e-12},
		{name: "Exp", fn: math.Exp, input: 2, want: math.E * math.E, tol: 1e-12},
		{name: "Exp2", fn: math.Exp2, input: 5, want: 32, tol: 1e-12},
		{name: "Expm1", fn: math.Expm1, input: 1, want: math.E - 1, tol: 1e-12},
		{name: "Floor", fn: math.Floor, input: -1.2, want: -2, tol: 0},
		{name: "Log", fn: math.Log, input: math.E, want: 1, tol: 1e-12},
		{name: "Log10", fn: math.Log10, input: 1000, want: 3, tol: 1e-12},
		{name: "Log1p", fn: math.Log1p, input: 1, want: math.Log(2), tol: 1e-12},
		{name: "Log2", fn: math.Log2, input: 32, want: 5, tol: 1e-12},
		{name: "Logb", fn: math.Logb, input: 8, want: 3, tol: 1e-12},
		{name: "Round", fn: math.Round, input: -1.5, want: -2, tol: 0},
		{name: "RoundToEven", fn: math.RoundToEven, input: 2.5, want: 2, tol: 0},
		{name: "Sin", fn: math.Sin, input: math.Pi / 2, want: 1, tol: 1e-12},
		{name: "Sinh", fn: math.Sinh, input: 1, want: (math.Exp(1) - math.Exp(-1)) / 2, tol: 1e-12},
		{name: "Sqrt", fn: math.Sqrt, input: 49, want: 7, tol: 1e-12},
		{name: "Tan", fn: math.Tan, input: math.Pi / 4, want: 1, tol: 1e-12},
		{name: "Tanh", fn: math.Tanh, input: 1, want: 0.7615941559557649, tol: 1e-12},
		{name: "Trunc", fn: math.Trunc, input: -1.9, want: -1, tol: 0},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := tc.fn(tc.input)
			assertFloatNear(t, got, tc.want, tc.tol)
		})
	}
}

func TestBinaryFloatFunctions(t *testing.T) {
	assertFloatNear(t, math.Atan2(1, -1), 3*math.Pi/4, 1e-12)

	if got := math.Copysign(3, -1); got != -3 {
		t.Fatalf("Copysign(3, -1) = %v, want -3", got)
	}
	if got := math.Copysign(-3, 1); got != 3 {
		t.Fatalf("Copysign(-3, 1) = %v, want 3", got)
	}

	if got := math.Dim(5, 3); got != 2 {
		t.Fatalf("Dim(5, 3) = %v, want 2", got)
	}
	if got := math.Dim(3, 5); got != 0 {
		t.Fatalf("Dim(3, 5) = %v, want 0", got)
	}

	assertFloatNear(t, math.Hypot(3, 4), 5, 1e-12)
	assertFloatNear(t, math.Max(1.2, 3.4), 3.4, 1e-12)
	assertFloatNear(t, math.Min(1.2, 3.4), 1.2, 1e-12)
	assertFloatNear(t, math.Pow(3, 4), 81, 1e-12)
	assertFloatNear(t, math.Remainder(5.5, 2), -0.5, 1e-12)
	assertFloatNear(t, math.FMA(2, 3, 4), 10, 1e-12)
}

func TestMod(t *testing.T) {
	testCases := []struct {
		x, y float64
		want float64
	}{
		{x: 10, y: 3, want: 1},
		{x: 10.5, y: 3, want: 1.5},
		{x: -10, y: 3, want: -1},
		{x: 10, y: -3, want: 1},
		{x: 7.5, y: 2.5, want: 0},
	}

	for _, tc := range testCases {
		got := math.Mod(tc.x, tc.y)
		assertFloatNear(t, got, tc.want, 1e-12)
	}
}

func TestPow10AndLdexp(t *testing.T) {
	assertFloatNear(t, math.Pow10(3), 1000, 0)

	if got := math.Ilogb(8); got != 3 {
		t.Fatalf("Ilogb(8) = %d, want 3", got)
	}

	assertFloatNear(t, math.Ldexp(0.75, 2), 3, 1e-12)
}

func TestFrexp(t *testing.T) {
	testCases := []struct {
		input    float64
		wantFrac float64
		wantExp  int
	}{
		{input: 0, wantFrac: 0, wantExp: 0},
		{input: 1, wantFrac: 0.5, wantExp: 1},
		{input: 8, wantFrac: 0.5, wantExp: 4},
		{input: -8, wantFrac: -0.5, wantExp: 4},
		{input: 0.25, wantFrac: 0.5, wantExp: -1},
	}

	for _, tc := range testCases {
		frac, exp := math.Frexp(tc.input)
		assertFloatNear(t, frac, tc.wantFrac, 0)
		if exp != tc.wantExp {
			t.Fatalf("Frexp(%v) exp = %v, want %v", tc.input, exp, tc.wantExp)
		}
	}
}

func TestModf(t *testing.T) {
	intPart, frac := math.Modf(3.5)
	assertFloatNear(t, intPart, 3, 0)
	assertFloatNear(t, frac, 0.5, 0)

	intPart, frac = math.Modf(-3.5)
	assertFloatNear(t, intPart, -3, 0)
	assertFloatNear(t, frac, -0.5, 0)
}

func TestLgamma(t *testing.T) {
	lgamma, sign := math.Lgamma(5)
	assertFloatNear(t, lgamma, 3.1780538303479458, 1e-12)
	if sign != 1 {
		t.Fatalf("Lgamma(5) sign = %d, want 1", sign)
	}
}

func TestGamma(t *testing.T) {
	assertFloatNear(t, math.Gamma(5), 24, 1e-12)
}

func TestSincos(t *testing.T) {
	s, c := math.Sincos(0.5)
	assertFloatNear(t, s, 0.479425538604203, 1e-12)
	assertFloatNear(t, c, 0.8775825618903728, 1e-12)
}

func TestFloatBitConversions(t *testing.T) {
	if got := math.Float32bits(3.5); got != 0x40600000 {
		t.Fatalf("Float32bits(3.5) = 0x%08x, want 0x40600000", got)
	}
	if got := math.Float32frombits(0x3f800000); got != 1 {
		t.Fatalf("Float32frombits(0x3f800000) = %v, want 1", got)
	}
	if got := math.Float64bits(3.5); got != 0x400c000000000000 {
		t.Fatalf("Float64bits(3.5) = 0x%x, want 0x400c000000000000", got)
	}
	if got := math.Float64frombits(0x3ff0000000000000); got != 1 {
		t.Fatalf("Float64frombits(0x3ff0000000000000) = %v, want 1", got)
	}
}

func TestFloatBitRoundTrips(t *testing.T) {
	bits32 := []uint32{0, 1, 0x007fffff, 0x00800000, 0x7f7fffff, 0x7f800000, 0xff800000,
		0x7fc12345, 0xffc12345, 0x7f800001, 0xff800001}
	for bit := uint(0); bit < 32; bit++ {
		bits32 = append(bits32, uint32(1)<<bit, ^(uint32(1) << bit))
	}
	for _, want := range bits32 {
		if got := math.Float32bits(math.Float32frombits(want)); got != want {
			t.Fatalf("Float32 round trip: %#08x != %#08x", got, want)
		}
	}
	bits64 := []uint64{0, 1, 0x000fffffffffffff, 0x0010000000000000, 0x7fefffffffffffff,
		0x7ff0000000000000, 0xfff0000000000000, 0x7ff8123456789abc, 0xfff8123456789abc,
		0x7ff0000000000001, 0xfff0000000000001}
	for bit := uint(0); bit < 64; bit++ {
		bits64 = append(bits64, uint64(1)<<bit, ^(uint64(1) << bit))
	}
	for _, want := range bits64 {
		if got := math.Float64bits(math.Float64frombits(want)); got != want {
			t.Fatalf("Float64 round trip: %#016x != %#016x", got, want)
		}
	}
}

// Taking an intrinsic's address still uses the ordinary function-value path.
// Keep that path working alongside direct caller-side lowering.
func TestFloatBitFunctionValues(t *testing.T) {
	from32 := []func(uint32) float32{math.Float32frombits}
	to32 := []func(float32) uint32{math.Float32bits}
	for _, want := range []uint32{0, 1, 0x80000000, 0x3f800000, 0x7f800000, 0x7fc12345} {
		if got := to32[0](from32[0](want)); got != want {
			t.Fatalf("Float32 function-value round trip: %#08x != %#08x", got, want)
		}
	}
	from64 := []func(uint64) float64{math.Float64frombits}
	to64 := []func(float64) uint64{math.Float64bits}
	for _, want := range []uint64{0, 1, 0x8000000000000000, 0x3ff0000000000000, 0x7ff0000000000000, 0x7ff8123456789abc} {
		if got := to64[0](from64[0](want)); got != want {
			t.Fatalf("Float64 function-value round trip: %#016x != %#016x", got, want)
		}
	}
}

func TestNextafter(t *testing.T) {
	next := math.Nextafter(1, 2)
	want := math.Float64frombits(math.Float64bits(1) + 1)
	assertFloatNear(t, next, want, 0)

	prev := math.Nextafter(1, 0)
	wantPrev := math.Float64frombits(math.Float64bits(1) - 1)
	assertFloatNear(t, prev, wantPrev, 0)

	next32 := math.Nextafter32(1, 2)
	want32 := math.Float32frombits(math.Float32bits(1) + 1)
	assertFloatNear(t, float64(next32), float64(want32), 0)

	prev32 := math.Nextafter32(1, 0)
	wantPrev32 := math.Float32frombits(math.Float32bits(1) - 1)
	assertFloatNear(t, float64(prev32), float64(wantPrev32), 0)
}

func TestNaNAndInf(t *testing.T) {
	if !math.IsNaN(math.NaN()) {
		t.Fatal("NaN() should return NaN")
	}

	if !math.IsInf(math.Inf(1), 1) {
		t.Fatal("Inf(1) should return positive infinity")
	}
	if !math.IsInf(math.Inf(-1), -1) {
		t.Fatal("Inf(-1) should return negative infinity")
	}

	if math.IsNaN(1.0) {
		t.Fatal("1.0 should not be NaN")
	}
	if math.IsInf(1.0, 0) {
		t.Fatal("1.0 should not be infinity")
	}
}

func TestSignbit(t *testing.T) {
	if !math.Signbit(-1) {
		t.Fatal("Signbit(-1) = false, want true")
	}
	if !math.Signbit(math.Copysign(0, -1)) {
		t.Fatal("Signbit(-0) = false, want true")
	}
	if math.Signbit(2) {
		t.Fatal("Signbit(2) = true, want false")
	}
}

func TestBesselFunctions(t *testing.T) {
	assertFloatNear(t, math.J0(5), -0.1775967713143383, 1e-12)
	assertFloatNear(t, math.J1(5), -0.3275791375914652, 1e-12)
	assertFloatNear(t, math.Jn(2, 5), 0.0465651162777522, 1e-12)

	assertFloatNear(t, math.Y0(5), -0.3085176252490338, 1e-12)
	assertFloatNear(t, math.Y1(5), 0.1478631433912268, 1e-12)
	assertFloatNear(t, math.Yn(2, 5), 0.3676628826055245, 1e-12)
}

func TestMathematicalConstants(t *testing.T) {
	tests := []struct {
		name string
		got  float64
		want float64
	}{
		{name: "E", got: math.E, want: math.Exp(1)},
		{name: "Pi", got: math.Pi, want: 4 * math.Atan(1)},
		{name: "Phi", got: math.Phi, want: (1 + math.Sqrt(5)) / 2},
		{name: "Sqrt2", got: math.Sqrt2, want: math.Sqrt(2)},
		{name: "SqrtE", got: math.SqrtE, want: math.Sqrt(math.Exp(1))},
		{name: "SqrtPi", got: math.SqrtPi, want: math.Sqrt(4 * math.Atan(1))},
		{name: "SqrtPhi", got: math.SqrtPhi, want: math.Sqrt((1 + math.Sqrt(5)) / 2)},
		{name: "Ln2", got: math.Ln2, want: math.Log(2)},
		{name: "Log2E", got: math.Log2E, want: 1 / math.Log(2)},
		{name: "Ln10", got: math.Ln10, want: math.Log(10)},
		{name: "Log10E", got: math.Log10E, want: 1 / math.Log(10)},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assertFloatNear(t, tc.got, tc.want, tolerance)
		})
	}
}

func TestFloatLimitConstants(t *testing.T) {
	if math.Float32bits(math.MaxFloat32) != 0x7f7fffff {
		t.Fatalf("Float32bits(MaxFloat32) = 0x%08x, want 0x7f7fffff", math.Float32bits(math.MaxFloat32))
	}
	if math.Float32bits(math.SmallestNonzeroFloat32) != 0x00000001 {
		t.Fatalf("Float32bits(SmallestNonzeroFloat32) = 0x%08x, want 0x00000001", math.Float32bits(math.SmallestNonzeroFloat32))
	}
	if math.Float64bits(math.MaxFloat64) != 0x7fefffffffffffff {
		t.Fatalf("Float64bits(MaxFloat64) = 0x%x, want 0x7fefffffffffffff", math.Float64bits(math.MaxFloat64))
	}
	if math.Float64bits(math.SmallestNonzeroFloat64) != 0x0000000000000001 {
		t.Fatalf("Float64bits(SmallestNonzeroFloat64) = 0x%x, want 0x0000000000000001", math.Float64bits(math.SmallestNonzeroFloat64))
	}
}

func TestIntegerLimitConstants(t *testing.T) {
	expectedMaxInt := int(^uint(0) >> 1)
	if int(math.MaxInt) != expectedMaxInt {
		t.Fatalf("MaxInt = %d, want %d", int(math.MaxInt), expectedMaxInt)
	}
	expectedMinInt := -expectedMaxInt - 1
	if int(math.MinInt) != expectedMinInt {
		t.Fatalf("MinInt = %d, want %d", int(math.MinInt), expectedMinInt)
	}
	if math.MaxInt8 != 1<<7-1 {
		t.Fatalf("MaxInt8 = %d, want %d", math.MaxInt8, 1<<7-1)
	}
	if math.MinInt8 != -1<<7 {
		t.Fatalf("MinInt8 = %d, want %d", math.MinInt8, -1<<7)
	}
	if math.MaxInt16 != 1<<15-1 {
		t.Fatalf("MaxInt16 = %d, want %d", math.MaxInt16, 1<<15-1)
	}
	if math.MinInt16 != -1<<15 {
		t.Fatalf("MinInt16 = %d, want %d", math.MinInt16, -1<<15)
	}
	if math.MaxInt32 != 1<<31-1 {
		t.Fatalf("MaxInt32 = %d, want %d", math.MaxInt32, 1<<31-1)
	}
	if math.MinInt32 != -1<<31 {
		t.Fatalf("MinInt32 = %d, want %d", math.MinInt32, -1<<31)
	}
	expectedMaxInt64 := int64(^uint64(0) >> 1)
	if int64(math.MaxInt64) != expectedMaxInt64 {
		t.Fatalf("MaxInt64 = %d, want %d", int64(math.MaxInt64), expectedMaxInt64)
	}
	expectedMinInt64 := -expectedMaxInt64 - 1
	if int64(math.MinInt64) != expectedMinInt64 {
		t.Fatalf("MinInt64 = %d, want %d", int64(math.MinInt64), expectedMinInt64)
	}
	if uint(math.MaxUint) != ^uint(0) {
		t.Fatalf("MaxUint = %d, want %d", uint64(math.MaxUint), uint64(^uint(0)))
	}
	if uint8(math.MaxUint8) != ^uint8(0) {
		t.Fatalf("MaxUint8 = %d, want %d", uint8(math.MaxUint8), ^uint8(0))
	}
	if uint16(math.MaxUint16) != ^uint16(0) {
		t.Fatalf("MaxUint16 = %d, want %d", uint16(math.MaxUint16), ^uint16(0))
	}
	if uint32(math.MaxUint32) != ^uint32(0) {
		t.Fatalf("MaxUint32 = %d, want %d", uint32(math.MaxUint32), ^uint32(0))
	}
	if uint64(math.MaxUint64) != ^uint64(0) {
		t.Fatalf("MaxUint64 = %d, want %d", uint64(math.MaxUint64), ^uint64(0))
	}
}
