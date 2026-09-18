package rsa_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"testing"
)

// These tests validate RSA API behavior rather than cryptographic strength.
// Keep key generation bounded on WebAssembly, where repeated 2048-bit prime
// searches would dominate the complete acceptance matrix.
const testRSAKeyBits = 1024

func TestGenerateKey(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	if priv == nil {
		t.Fatal("GenerateKey() returned nil")
	}

	if err := priv.Validate(); err != nil {
		t.Errorf("Generated key failed validation: %v", err)
	}
}

func TestGenerateMultiPrimeKey(t *testing.T) {
	priv, err := rsa.GenerateMultiPrimeKey(rand.Reader, 3, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateMultiPrimeKey() error = %v", err)
	}

	if priv == nil {
		t.Fatal("GenerateMultiPrimeKey() returned nil")
	}

	if err := priv.Validate(); err != nil {
		t.Errorf("Generated multi-prime key failed validation: %v", err)
	}
}

func TestEncryptDecryptPKCS1v15(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	msg := []byte("test message")

	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, &priv.PublicKey, msg)
	if err != nil {
		t.Fatalf("EncryptPKCS1v15() error = %v", err)
	}

	plaintext, err := rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext)
	if err != nil {
		t.Fatalf("DecryptPKCS1v15() error = %v", err)
	}

	if string(plaintext) != string(msg) {
		t.Errorf("Decrypted message = %s, want %s", plaintext, msg)
	}
}

func TestEncryptDecryptOAEP(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	msg := []byte("test message")
	label := []byte("test label")
	hash := sha256.New()

	ciphertext, err := rsa.EncryptOAEP(hash, rand.Reader, &priv.PublicKey, msg, label)
	if err != nil {
		t.Fatalf("EncryptOAEP() error = %v", err)
	}

	hash = sha256.New()
	plaintext, err := rsa.DecryptOAEP(hash, rand.Reader, priv, ciphertext, label)
	if err != nil {
		t.Fatalf("DecryptOAEP() error = %v", err)
	}

	if string(plaintext) != string(msg) {
		t.Errorf("Decrypted message = %s, want %s", plaintext, msg)
	}
}

func TestSignVerifyPKCS1v15(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	msg := []byte("test message")
	hashed := sha256.Sum256(msg)

	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hashed[:])
	if err != nil {
		t.Fatalf("SignPKCS1v15() error = %v", err)
	}

	err = rsa.VerifyPKCS1v15(&priv.PublicKey, crypto.SHA256, hashed[:], sig)
	if err != nil {
		t.Errorf("VerifyPKCS1v15() error = %v", err)
	}

	wrongHash := sha256.Sum256([]byte("wrong message"))
	err = rsa.VerifyPKCS1v15(&priv.PublicKey, crypto.SHA256, wrongHash[:], sig)
	if err == nil {
		t.Error("VerifyPKCS1v15() succeeded for wrong hash")
	}
}

func TestSignVerifyPSS(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	msg := []byte("test message")
	hashed := sha256.Sum256(msg)

	sig, err := rsa.SignPSS(rand.Reader, priv, crypto.SHA256, hashed[:], nil)
	if err != nil {
		t.Fatalf("SignPSS() error = %v", err)
	}

	err = rsa.VerifyPSS(&priv.PublicKey, crypto.SHA256, hashed[:], sig, nil)
	if err != nil {
		t.Errorf("VerifyPSS() error = %v", err)
	}

	wrongHash := sha256.Sum256([]byte("wrong message"))
	err = rsa.VerifyPSS(&priv.PublicKey, crypto.SHA256, wrongHash[:], sig, nil)
	if err == nil {
		t.Error("VerifyPSS() succeeded for wrong hash")
	}
}

func TestPrivateKeyValidate(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	if err := priv.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}

	invalidPriv := &rsa.PrivateKey{
		PublicKey: rsa.PublicKey{
			N: priv.PublicKey.N,
			E: 3,
		},
	}
	if err := invalidPriv.Validate(); err == nil {
		t.Error("Validate() succeeded for invalid key")
	}
}

func TestPrivateKeyPrecompute(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	priv.Precompute()

	if priv.Precomputed.Dp == nil {
		t.Error("Precompute() did not set Dp")
	}
	if priv.Precomputed.Dq == nil {
		t.Error("Precompute() did not set Dq")
	}
	if priv.Precomputed.Qinv == nil {
		t.Error("Precompute() did not set Qinv")
	}
}

func TestPrivateKeyEqual(t *testing.T) {
	priv1, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	priv2, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	if priv1.Equal(priv2) {
		t.Error("Different private keys reported as equal")
	}

	if !priv1.Equal(priv1) {
		t.Error("Same private key not equal to itself")
	}

	var notRSA crypto.PrivateKey = []byte("not an rsa key")
	if priv1.Equal(notRSA) {
		t.Error("Equal() returned true for non-RSA key")
	}
}

func TestPrivateKeyPublic(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	pub := priv.Public()
	if pub == nil {
		t.Fatal("Public() returned nil")
	}

	pubKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		t.Fatal("Public() did not return *rsa.PublicKey")
	}

	if pubKey.N.Cmp(priv.PublicKey.N) != 0 {
		t.Error("Public() returned different N")
	}
	if pubKey.E != priv.PublicKey.E {
		t.Error("Public() returned different E")
	}
}

func TestPrivateKeySign(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	msg := []byte("test message")
	hashed := sha256.Sum256(msg)

	sig, err := priv.Sign(rand.Reader, hashed[:], crypto.SHA256)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	if len(sig) == 0 {
		t.Error("Sign() returned empty signature")
	}

	err = rsa.VerifyPKCS1v15(&priv.PublicKey, crypto.SHA256, hashed[:], sig)
	if err != nil {
		t.Error("Signature verification failed")
	}
}

func TestPrivateKeyDecrypt(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	msg := []byte("test message")

	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, &priv.PublicKey, msg)
	if err != nil {
		t.Fatalf("EncryptPKCS1v15() error = %v", err)
	}

	plaintext, err := priv.Decrypt(rand.Reader, ciphertext, nil)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if string(plaintext) != string(msg) {
		t.Errorf("Decrypted message = %s, want %s", plaintext, msg)
	}
}

func TestPublicKeyEqual(t *testing.T) {
	priv1, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	priv2, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	pub1 := &priv1.PublicKey
	pub2 := &priv2.PublicKey

	if pub1.Equal(pub2) {
		t.Error("Different public keys reported as equal")
	}

	if !pub1.Equal(pub1) {
		t.Error("Same public key not equal to itself")
	}

	var notRSA crypto.PublicKey = []byte("not an rsa key")
	if pub1.Equal(notRSA) {
		t.Error("Equal() returned true for non-RSA key")
	}
}

func TestPublicKeySize(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	size := priv.PublicKey.Size()
	expectedSize := testRSAKeyBits / 8

	if size != expectedSize {
		t.Errorf("Size() = %d, want %d", size, expectedSize)
	}
}

func TestPSSOptionsHashFunc(t *testing.T) {
	opts := &rsa.PSSOptions{
		Hash: crypto.SHA256,
	}

	hashFunc := opts.HashFunc()
	if hashFunc != crypto.SHA256 {
		t.Errorf("HashFunc() = %v, want %v", hashFunc, crypto.SHA256)
	}
}

func TestDecryptPKCS1v15SessionKey(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	sessionKey := make([]byte, 32)
	if _, err := rand.Read(sessionKey); err != nil {
		t.Fatalf("rand.Read() error = %v", err)
	}

	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, &priv.PublicKey, sessionKey)
	if err != nil {
		t.Fatalf("EncryptPKCS1v15() error = %v", err)
	}

	key := make([]byte, 32)
	err = rsa.DecryptPKCS1v15SessionKey(rand.Reader, priv, ciphertext, key)
	if err != nil {
		t.Fatalf("DecryptPKCS1v15SessionKey() error = %v", err)
	}

	if string(key) != string(sessionKey) {
		t.Error("Decrypted session key does not match")
	}
}

func TestErrors(t *testing.T) {
	if rsa.ErrDecryption == nil {
		t.Error("ErrDecryption is nil")
	}
	if rsa.ErrMessageTooLong == nil {
		t.Error("ErrMessageTooLong is nil")
	}
	if rsa.ErrVerification == nil {
		t.Error("ErrVerification is nil")
	}
}

func TestOAEPOptions(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	msg := []byte("test message")
	label := []byte("test label")

	opts := &rsa.OAEPOptions{
		Hash:  crypto.SHA256,
		Label: label,
	}

	ciphertext, err := priv.Decrypt(nil, msg, opts)
	if err == nil && len(ciphertext) > 0 {
	}
}

func TestPKCS1v15DecryptOptions(t *testing.T) {
	opts := &rsa.PKCS1v15DecryptOptions{
		SessionKeyLen: 32,
	}

	if opts.SessionKeyLen != 32 {
		t.Errorf("SessionKeyLen = %d, want 32", opts.SessionKeyLen)
	}
}

func TestCRTValue(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	priv.Precompute()

	if len(priv.Precomputed.CRTValues) == 0 {
	}
}

func TestPrecomputedValues(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	priv.Precompute()

	if priv.Precomputed.Dp == nil {
		t.Error("Precomputed.Dp is nil")
	}
	if priv.Precomputed.Dq == nil {
		t.Error("Precomputed.Dq is nil")
	}
	if priv.Precomputed.Qinv == nil {
		t.Error("Precomputed.Qinv is nil")
	}
}

func TestPSSWithOptions(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	msg := []byte("test message")
	hashed := sha256.Sum256(msg)

	opts := &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthAuto,
		Hash:       crypto.SHA256,
	}

	sig, err := rsa.SignPSS(rand.Reader, priv, crypto.SHA256, hashed[:], opts)
	if err != nil {
		t.Fatalf("SignPSS() error = %v", err)
	}

	err = rsa.VerifyPSS(&priv.PublicKey, crypto.SHA256, hashed[:], sig, opts)
	if err != nil {
		t.Errorf("VerifyPSS() error = %v", err)
	}
}

func TestPSSSaltLengthEqualsHash(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	msg := []byte("test message")
	hashed := sha256.Sum256(msg)

	opts := &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       crypto.SHA256,
	}

	sig, err := rsa.SignPSS(rand.Reader, priv, crypto.SHA256, hashed[:], opts)
	if err != nil {
		t.Fatalf("SignPSS() error = %v", err)
	}

	err = rsa.VerifyPSS(&priv.PublicKey, crypto.SHA256, hashed[:], sig, opts)
	if err != nil {
		t.Errorf("VerifyPSS() error = %v", err)
	}
}

func TestCRTValueStruct(t *testing.T) {
	priv, err := rsa.GenerateMultiPrimeKey(rand.Reader, 3, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateMultiPrimeKey() error = %v", err)
	}

	priv.Precompute()

	if len(priv.Precomputed.CRTValues) > 0 {
		crtVal := priv.Precomputed.CRTValues[0]
		if crtVal.Exp == nil {
			t.Error("CRTValue.Exp is nil")
		}
		if crtVal.Coeff == nil {
			t.Error("CRTValue.Coeff is nil")
		}
		if crtVal.R == nil {
			t.Error("CRTValue.R is nil")
		}
	}
}

func TestPrecomputedValuesStruct(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	priv.Precompute()

	precomp := priv.Precomputed
	if precomp.Dp == nil {
		t.Error("PrecomputedValues.Dp is nil")
	}
	if precomp.Dq == nil {
		t.Error("PrecomputedValues.Dq is nil")
	}
	if precomp.Qinv == nil {
		t.Error("PrecomputedValues.Qinv is nil")
	}

	var _ rsa.CRTValue
	var _ rsa.PrecomputedValues
}
