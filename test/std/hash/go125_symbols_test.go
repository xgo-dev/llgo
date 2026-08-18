//go:build go1.25

package hash_test

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha3"
	"hash"
	"io"
	"testing"
)

func TestCloner(t *testing.T) {
	var original hash.Cloner = sha256.New().(hash.Cloner)
	original.Write([]byte("prefix"))
	clone, err := original.Clone()
	if err != nil {
		t.Fatal(err)
	}
	original.Write([]byte("same"))
	clone.Write([]byte("same"))
	if !bytes.Equal(original.Sum(nil), clone.Sum(nil)) {
		t.Fatal("clone did not preserve hash state")
	}
	clone.Write([]byte("different"))
	if bytes.Equal(original.Sum(nil), clone.Sum(nil)) {
		t.Fatal("clone state is not independent")
	}
}

func TestXOF(t *testing.T) {
	var xof hash.XOF = sha3.NewSHAKE128()
	if xof.BlockSize() <= 0 {
		t.Fatalf("BlockSize = %d", xof.BlockSize())
	}
	xof.Write([]byte("llgo"))
	first := make([]byte, 32)
	if _, err := io.ReadFull(xof, first); err != nil {
		t.Fatal(err)
	}
	xof.Reset()
	xof.Write([]byte("llgo"))
	second := make([]byte, len(first))
	if _, err := io.ReadFull(xof, second); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("Reset did not restore the XOF state")
	}
}
