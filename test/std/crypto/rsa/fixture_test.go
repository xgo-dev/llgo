package rsa_test

import (
	"crypto/rsa"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"errors"
)

// These test-only keys let operation tests avoid repeating RSA prime searches.
// TestGenerateKey and TestGenerateMultiPrimeKey still exercise real generation.
//
//go:embed testdata/key1.pem
var key1PEM []byte

//go:embed testdata/key2.pem
var key2PEM []byte

//go:embed testdata/key3.pem
var key3PEM []byte

func fixtureKey(index int) (*rsa.PrivateKey, error) {
	var data []byte
	switch index {
	case 1:
		data = key1PEM
	case 2:
		data = key2PEM
	case 3:
		data = key3PEM
	default:
		return nil, errors.New("invalid RSA fixture index")
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid RSA fixture PEM")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
