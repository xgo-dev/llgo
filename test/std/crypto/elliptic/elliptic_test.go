package elliptic_test

import (
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"testing"
)

func TestP224(t *testing.T) {
	curve := elliptic.P224()
	if curve == nil {
		t.Fatal("P224() returned nil")
	}
	params := curve.Params()
	if params == nil {
		t.Fatal("Params() returned nil")
	}
	if params.BitSize != 224 {
		t.Errorf("P224 BitSize = %d, want 224", params.BitSize)
	}
}

func TestP256(t *testing.T) {
	curve := elliptic.P256()
	if curve == nil {
		t.Fatal("P256() returned nil")
	}
	params := curve.Params()
	if params == nil {
		t.Fatal("Params() returned nil")
	}
	if params.BitSize != 256 {
		t.Errorf("P256 BitSize = %d, want 256", params.BitSize)
	}
}

func TestP384(t *testing.T) {
	curve := elliptic.P384()
	if curve == nil {
		t.Fatal("P384() returned nil")
	}
	params := curve.Params()
	if params == nil {
		t.Fatal("Params() returned nil")
	}
	if params.BitSize != 384 {
		t.Errorf("P384 BitSize = %d, want 384", params.BitSize)
	}
}

func TestP521(t *testing.T) {
	curve := elliptic.P521()
	if curve == nil {
		t.Fatal("P521() returned nil")
	}
	params := curve.Params()
	if params == nil {
		t.Fatal("Params() returned nil")
	}
	if params.BitSize != 521 {
		t.Errorf("P521 BitSize = %d, want 521", params.BitSize)
	}
}

func TestGenerateKey(t *testing.T) {
	curves := []struct {
		name  string
		curve elliptic.Curve
	}{
		{"P224", elliptic.P224()},
		{"P256", elliptic.P256()},
		{"P384", elliptic.P384()},
		{"P521", elliptic.P521()},
	}

	for _, tc := range curves {
		t.Run(tc.name, func(t *testing.T) {
			priv, x, y, err := elliptic.GenerateKey(tc.curve, rand.Reader)
			if err != nil {
				t.Fatalf("GenerateKey() error = %v", err)
			}
			if len(priv) == 0 {
				t.Error("GenerateKey() returned empty private key")
			}
			if x == nil || y == nil {
				t.Fatal("GenerateKey() returned nil coordinates")
			}
			if !tc.curve.IsOnCurve(x, y) {
				t.Error("Generated point is not on curve")
			}
		})
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	curve := elliptic.P256()
	_, x, y, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	data := elliptic.Marshal(curve, x, y)
	if len(data) == 0 {
		t.Fatal("Marshal() returned empty data")
	}

	x2, y2 := elliptic.Unmarshal(curve, data)
	if x2 == nil || y2 == nil {
		t.Fatal("Unmarshal() returned nil coordinates")
	}

	if x.Cmp(x2) != 0 || y.Cmp(y2) != 0 {
		t.Error("Unmarshal() did not restore original coordinates")
	}
}

func TestMarshalCompressedUnmarshalCompressed(t *testing.T) {
	curve := elliptic.P256()
	_, x, y, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	data := elliptic.MarshalCompressed(curve, x, y)
	if len(data) == 0 {
		t.Fatal("MarshalCompressed() returned empty data")
	}

	uncompressedData := elliptic.Marshal(curve, x, y)
	if len(data) >= len(uncompressedData) {
		t.Error("MarshalCompressed() data is not smaller than uncompressed")
	}

	x2, y2 := elliptic.UnmarshalCompressed(curve, data)
	if x2 == nil || y2 == nil {
		t.Fatal("UnmarshalCompressed() returned nil coordinates")
	}

	if x.Cmp(x2) != 0 || y.Cmp(y2) != 0 {
		t.Error("UnmarshalCompressed() did not restore original coordinates")
	}
}

func TestCurveParamsIsOnCurve(t *testing.T) {
	curve := elliptic.P256()
	_, x, y, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	if !curve.IsOnCurve(x, y) {
		t.Error("IsOnCurve() returned false for valid point")
	}

	badX := new(big.Int).Add(x, big.NewInt(1))
	if curve.IsOnCurve(badX, y) {
		t.Error("IsOnCurve() returned true for invalid point")
	}
}

func TestCurveParamsAdd(t *testing.T) {
	curve := elliptic.P256()
	_, x1, y1, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	_, x2, y2, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	x3, y3 := curve.Add(x1, y1, x2, y2)
	if x3 == nil || y3 == nil {
		t.Fatal("Add() returned nil coordinates")
	}

	if !curve.IsOnCurve(x3, y3) {
		t.Error("Add() result is not on curve")
	}
}

func TestCurveParamsDouble(t *testing.T) {
	curve := elliptic.P256()
	_, x, y, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	x2, y2 := curve.Double(x, y)
	if x2 == nil || y2 == nil {
		t.Fatal("Double() returned nil coordinates")
	}

	if !curve.IsOnCurve(x2, y2) {
		t.Error("Double() result is not on curve")
	}

	x3, y3 := curve.Add(x, y, x, y)
	if x2.Cmp(x3) != 0 || y2.Cmp(y3) != 0 {
		t.Error("Double() does not match Add(P, P)")
	}
}

func TestCurveParamsScalarBaseMult(t *testing.T) {
	curve := elliptic.P256()

	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		t.Fatalf("rand.Read() error = %v", err)
	}

	x, y := curve.ScalarBaseMult(k)
	if x == nil || y == nil {
		t.Fatal("ScalarBaseMult() returned nil coordinates")
	}

	if !curve.IsOnCurve(x, y) {
		t.Error("ScalarBaseMult() result is not on curve")
	}

	zeroK := make([]byte, 32)
	x0, y0 := curve.ScalarBaseMult(zeroK)
	if x0.Sign() != 0 || y0.Sign() != 0 {
		t.Error("ScalarBaseMult(0) should return point at infinity")
	}
}

func TestCurveParamsScalarMult(t *testing.T) {
	curve := elliptic.P256()
	_, bx, by, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		t.Fatalf("rand.Read() error = %v", err)
	}

	x, y := curve.ScalarMult(bx, by, k)
	if x == nil || y == nil {
		t.Fatal("ScalarMult() returned nil coordinates")
	}

	if !curve.IsOnCurve(x, y) {
		t.Error("ScalarMult() result is not on curve")
	}

	zeroK := make([]byte, 32)
	x0, y0 := curve.ScalarMult(bx, by, zeroK)
	if x0.Sign() != 0 || y0.Sign() != 0 {
		t.Error("ScalarMult(P, 0) should return point at infinity")
	}
}

func TestCurveParamsParams(t *testing.T) {
	curve := elliptic.P256()
	params := curve.Params()
	if params == nil {
		t.Fatal("Params() returned nil")
	}

	params2 := params.Params()
	if params2 != params {
		t.Error("Params().Params() did not return same object")
	}

	if params.P == nil {
		t.Error("Params().P is nil")
	}
	if params.N == nil {
		t.Error("Params().N is nil")
	}
	if params.B == nil {
		t.Error("Params().B is nil")
	}
	if params.Gx == nil {
		t.Error("Params().Gx is nil")
	}
	if params.Gy == nil {
		t.Error("Params().Gy is nil")
	}
}

func TestAllCurvesBasicOperations(t *testing.T) {
	curves := []struct {
		name  string
		curve elliptic.Curve
	}{
		{"P224", elliptic.P224()},
		{"P256", elliptic.P256()},
		{"P384", elliptic.P384()},
		{"P521", elliptic.P521()},
	}

	for _, tc := range curves {
		t.Run(tc.name, func(t *testing.T) {
			curve := tc.curve

			priv, x, y, err := elliptic.GenerateKey(curve, rand.Reader)
			if err != nil {
				t.Fatalf("GenerateKey() error = %v", err)
			}

			if !curve.IsOnCurve(x, y) {
				t.Error("Generated point not on curve")
			}

			data := elliptic.Marshal(curve, x, y)
			x2, y2 := elliptic.Unmarshal(curve, data)
			if x.Cmp(x2) != 0 || y.Cmp(y2) != 0 {
				t.Error("Marshal/Unmarshal round trip failed")
			}

			compData := elliptic.MarshalCompressed(curve, x, y)
			x3, y3 := elliptic.UnmarshalCompressed(curve, compData)
			if x.Cmp(x3) != 0 || y.Cmp(y3) != 0 {
				t.Error("MarshalCompressed/UnmarshalCompressed round trip failed")
			}

			xBase, yBase := curve.ScalarBaseMult(priv)
			if x.Cmp(xBase) != 0 || y.Cmp(yBase) != 0 {
				t.Error("ScalarBaseMult did not reproduce public key")
			}
		})
	}
}

func TestInvalidUnmarshal(t *testing.T) {
	curve := elliptic.P256()

	invalidData := []byte{0x04}
	x, y := elliptic.Unmarshal(curve, invalidData)
	if x != nil || y != nil {
		t.Error("Unmarshal() should return nil for invalid data")
	}

	emptyData := []byte{}
	x, y = elliptic.Unmarshal(curve, emptyData)
	if x != nil || y != nil {
		t.Error("Unmarshal() should return nil for empty data")
	}
}

func TestInvalidUnmarshalCompressed(t *testing.T) {
	curve := elliptic.P256()

	invalidData := []byte{0x02}
	x, y := elliptic.UnmarshalCompressed(curve, invalidData)
	if x != nil || y != nil {
		t.Error("UnmarshalCompressed() should return nil for invalid data")
	}

	emptyData := []byte{}
	x, y = elliptic.UnmarshalCompressed(curve, emptyData)
	if x != nil || y != nil {
		t.Error("UnmarshalCompressed() should return nil for empty data")
	}
}

func TestCurveParamsDirectMethods(t *testing.T) {
	params := elliptic.P256().Params()

	_, x1, y1, err := elliptic.GenerateKey(params, rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	if !params.IsOnCurve(x1, y1) {
		t.Error("CurveParams.IsOnCurve() returned false for valid point")
	}

	_, x2, y2, err := elliptic.GenerateKey(params, rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	x3, y3 := params.Add(x1, y1, x2, y2)
	if !params.IsOnCurve(x3, y3) {
		t.Error("CurveParams.Add() result is not on curve")
	}

	x4, y4 := params.Double(x1, y1)
	if !params.IsOnCurve(x4, y4) {
		t.Error("CurveParams.Double() result is not on curve")
	}

	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		t.Fatalf("rand.Read() error = %v", err)
	}

	x5, y5 := params.ScalarBaseMult(k)
	if !params.IsOnCurve(x5, y5) {
		t.Error("CurveParams.ScalarBaseMult() result is not on curve")
	}

	x6, y6 := params.ScalarMult(x1, y1, k)
	if !params.IsOnCurve(x6, y6) {
		t.Error("CurveParams.ScalarMult() result is not on curve")
	}
}
