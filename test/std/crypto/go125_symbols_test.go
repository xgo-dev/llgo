//go:build go1.25

package crypto_test

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"io"
	"testing"
)

type messageSigner struct {
	key ed25519.PrivateKey
}

func (s messageSigner) Public() crypto.PublicKey {
	return s.key.Public()
}

func (s messageSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return s.key.Sign(rand, digest, opts)
}

func (s messageSigner) SignMessage(_ io.Reader, message []byte, _ crypto.SignerOpts) ([]byte, error) {
	return ed25519.Sign(s.key, message), nil
}

func TestMessageSigner(t *testing.T) {
	seed := bytes.Repeat([]byte{0x42}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	var signer crypto.MessageSigner = messageSigner{key: privateKey}
	message := []byte("message signer")
	signature, err := crypto.SignMessage(signer, nil, message, crypto.Hash(0))
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(privateKey.Public().(ed25519.PublicKey), message, signature) {
		t.Fatal("SignMessage produced an invalid signature")
	}
}
