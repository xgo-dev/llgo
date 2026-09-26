//go:build go1.26

package rsa_test

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	_ "crypto/sha1"
	_ "crypto/sha256"
	"testing"
)

func TestEncryptOAEPWithOptions(t *testing.T) {
	key, err := fixtureKey(1)
	if err != nil {
		t.Fatal(err)
	}
	options := &rsa.OAEPOptions{
		Hash:    crypto.SHA256,
		MGFHash: crypto.SHA1,
		Label:   []byte("llgo"),
	}
	message := []byte("go1.26 OAEP options")
	ciphertext, err := rsa.EncryptOAEPWithOptions(rand.Reader, &key.PublicKey, message, options)
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := key.Decrypt(nil, ciphertext, options)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plaintext, message) {
		t.Fatalf("decrypted plaintext = %q, want %q", plaintext, message)
	}
}
