package dsa_test

import (
	"bytes"
	"crypto/dsa"
	"crypto/rand"
	"crypto/sha256"
	"math/big"
	"testing"
)

// Generating DSA parameters is deliberately covered by TestGenerateParameters.
// Reuse a valid set for the key and signature tests so those tests measure the
// operations they exercise instead of repeating an expensive prime search.
func testParameters(t *testing.T) dsa.Parameters {
	t.Helper()
	parse := func(hex string) *big.Int {
		n, ok := new(big.Int).SetString(hex, 16)
		if !ok {
			t.Fatal("invalid DSA test parameter")
		}
		return n
	}
	return dsa.Parameters{
		P: parse("d16e8c5d23efac0f2ca8f0a5dbd78984c7d684e99202244097d8ea0db4af099a0516fe4b87ca656a60f9710971279fbc398c63fa4ccd8a031f21a8e8e859eaf490112dac68380ffa6df3794d247c7d1e821d09138f27d83ad143159b163d502562de4dce936a9b4a2f8a089f36c2d68779b5c0a3b6ed8f66ce9e76c059a634c5"),
		Q: parse("811befff8b7132daca90988be96b31aaaa5c5a95"),
		G: parse("86d7c83ad34f639f3dbd8cab8bec284b00bf84c67ceb32a7ff0b2cfd750e47026b39e761bde4e468a9cd173e3eeeff71788c2884e9b34abe9ebc799b0b51f7fe9c715166d9eae1464bb64787db1efa5054669999f72861f204d2732a4fc789823d81cb558106812456964e662e70d39fd4f6f52e8dffc507bff5369d2cc86335"),
	}
}

func TestParameterSizes(t *testing.T) {
	tests := []struct {
		name string
		size dsa.ParameterSizes
	}{
		{"L1024N160", dsa.L1024N160},
		{"L2048N224", dsa.L2048N224},
		{"L2048N256", dsa.L2048N256},
		{"L3072N256", dsa.L3072N256},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.size < 0 {
				t.Errorf("ParameterSizes %s has negative value", tt.name)
			}
		})
	}
}

func TestGenerateParameters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping parameter generation in short mode")
	}

	// Feed valid prime candidates so this checks parameter construction without
	// depending on the unbounded random search for primes in an interpreter.
	candidates := testParameters(t)
	input := append(candidates.Q.Bytes(), candidates.P.Bytes()...)
	var params dsa.Parameters
	err := dsa.GenerateParameters(&params, bytes.NewReader(input), dsa.L1024N160)
	if err != nil {
		t.Fatalf("GenerateParameters() error = %v", err)
	}

	if params.P == nil || params.Q == nil || params.G == nil {
		t.Error("GenerateParameters() returned nil parameters")
	}

	if params.P.BitLen() != 1024 {
		t.Errorf("P bit length = %d, want 1024", params.P.BitLen())
	}

	if params.Q.BitLen() != 160 {
		t.Errorf("Q bit length = %d, want 160", params.Q.BitLen())
	}
}

func TestGenerateKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping key generation in short mode")
	}

	params := testParameters(t)

	var priv dsa.PrivateKey
	priv.Parameters = params
	err := dsa.GenerateKey(&priv, rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	if priv.X == nil {
		t.Error("GenerateKey() X is nil")
	}

	if priv.Y == nil {
		t.Error("GenerateKey() Y is nil")
	}

	if priv.X.Sign() <= 0 {
		t.Error("GenerateKey() X is not positive")
	}

	if priv.Y.Sign() <= 0 {
		t.Error("GenerateKey() Y is not positive")
	}
}

func TestSignAndVerify(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sign/verify test in short mode")
	}

	params := testParameters(t)

	var priv dsa.PrivateKey
	priv.Parameters = params
	err := dsa.GenerateKey(&priv, rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	message := []byte("test message")
	hash := sha256.Sum256(message)

	r, s, err := dsa.Sign(rand.Reader, &priv, hash[:])
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	if r == nil || s == nil {
		t.Fatal("Sign() returned nil signature components")
	}

	if r.Sign() <= 0 || s.Sign() <= 0 {
		t.Error("Sign() returned non-positive signature components")
	}

	pub := &priv.PublicKey
	if !dsa.Verify(pub, hash[:], r, s) {
		t.Error("Verify() failed for valid signature")
	}

	wrongHash := sha256.Sum256([]byte("wrong message"))
	if dsa.Verify(pub, wrongHash[:], r, s) {
		t.Error("Verify() succeeded for wrong hash")
	}

	wrongR := new(big.Int).Add(r, big.NewInt(1))
	if dsa.Verify(pub, hash[:], wrongR, s) {
		t.Error("Verify() succeeded for wrong r")
	}

	wrongS := new(big.Int).Add(s, big.NewInt(1))
	if dsa.Verify(pub, hash[:], r, wrongS) {
		t.Error("Verify() succeeded for wrong s")
	}
}

func TestErrInvalidPublicKey(t *testing.T) {
	if dsa.ErrInvalidPublicKey == nil {
		t.Error("ErrInvalidPublicKey is nil")
	}

	expectedMsg := "crypto/dsa: invalid public key"
	if dsa.ErrInvalidPublicKey.Error() != expectedMsg {
		t.Errorf("ErrInvalidPublicKey.Error() = %q, want %q",
			dsa.ErrInvalidPublicKey.Error(), expectedMsg)
	}
}

func TestParameters(t *testing.T) {
	params := dsa.Parameters{
		P: big.NewInt(123),
		Q: big.NewInt(456),
		G: big.NewInt(789),
	}

	if params.P.Int64() != 123 {
		t.Errorf("Parameters.P = %v, want 123", params.P)
	}

	if params.Q.Int64() != 456 {
		t.Errorf("Parameters.Q = %v, want 456", params.Q)
	}

	if params.G.Int64() != 789 {
		t.Errorf("Parameters.G = %v, want 789", params.G)
	}
}

func TestPublicKey(t *testing.T) {
	pub := dsa.PublicKey{
		Parameters: dsa.Parameters{
			P: big.NewInt(123),
			Q: big.NewInt(456),
			G: big.NewInt(789),
		},
		Y: big.NewInt(111),
	}

	if pub.Y.Int64() != 111 {
		t.Errorf("PublicKey.Y = %v, want 111", pub.Y)
	}

	if pub.Parameters.P.Int64() != 123 {
		t.Errorf("PublicKey.Parameters.P = %v, want 123", pub.Parameters.P)
	}
}

func TestPrivateKey(t *testing.T) {
	priv := dsa.PrivateKey{
		PublicKey: dsa.PublicKey{
			Parameters: dsa.Parameters{
				P: big.NewInt(123),
				Q: big.NewInt(456),
				G: big.NewInt(789),
			},
			Y: big.NewInt(111),
		},
		X: big.NewInt(222),
	}

	if priv.X.Int64() != 222 {
		t.Errorf("PrivateKey.X = %v, want 222", priv.X)
	}

	if priv.PublicKey.Y.Int64() != 111 {
		t.Errorf("PrivateKey.PublicKey.Y = %v, want 111", priv.PublicKey.Y)
	}

	if priv.PublicKey.Parameters.P.Int64() != 123 {
		t.Errorf("PrivateKey.PublicKey.Parameters.P = %v, want 123", priv.PublicKey.Parameters.P)
	}
}
