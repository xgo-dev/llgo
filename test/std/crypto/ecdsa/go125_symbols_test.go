//go:build go1.25

package ecdsa_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
)

func TestKeyBytes(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	privateBytes, err := privateKey.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if len(privateBytes) != 32 {
		t.Fatalf("PrivateKey.Bytes length = %d, want 32", len(privateBytes))
	}
	parsedPrivate, err := ecdsa.ParseRawPrivateKey(elliptic.P256(), privateBytes)
	if err != nil {
		t.Fatal(err)
	}
	if parsedPrivate.D.Cmp(privateKey.D) != 0 {
		t.Fatal("parsed private key differs from generated key")
	}

	publicBytes, err := privateKey.PublicKey.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	wantPublic := elliptic.Marshal(elliptic.P256(), privateKey.X, privateKey.Y)
	if !bytes.Equal(publicBytes, wantPublic) {
		t.Fatalf("PublicKey.Bytes = %x, want %x", publicBytes, wantPublic)
	}
	parsedPublic, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), publicBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !parsedPublic.Equal(&privateKey.PublicKey) {
		t.Fatal("parsed public key differs from generated key")
	}
}
