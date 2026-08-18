//go:build go1.26

package crypto_test

import (
	"bytes"
	"crypto"
	"crypto/mlkem"
	"testing"
)

func TestGo126EncapsulationInterfaces(t *testing.T) {
	key, err := mlkem.GenerateKey768()
	if err != nil {
		t.Fatal(err)
	}
	var decapsulator crypto.Decapsulator = key
	var encapsulator crypto.Encapsulator = decapsulator.Encapsulator()
	if !bytes.Equal(encapsulator.Bytes(), key.EncapsulationKey().Bytes()) {
		t.Fatal("Encapsulator returned the wrong public key")
	}
	shared, ciphertext := encapsulator.Encapsulate()
	decapsulated, err := decapsulator.Decapsulate(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(shared, decapsulated) {
		t.Fatal("encapsulated and decapsulated shared keys differ")
	}
}
