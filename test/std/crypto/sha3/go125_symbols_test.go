//go:build go1.25

package sha3_test

import (
	"bytes"
	"crypto/sha3"
	"testing"
)

func TestClone(t *testing.T) {
	h := sha3.New256()
	h.Write([]byte("prefix:"))
	clone, err := h.Clone()
	if err != nil {
		t.Fatal(err)
	}
	h.Write([]byte("same"))
	clone.Write([]byte("same"))
	if !bytes.Equal(h.Sum(nil), clone.Sum(nil)) {
		t.Fatal("clone did not preserve hash state")
	}
	clone.Write([]byte(":different"))
	if bytes.Equal(h.Sum(nil), clone.Sum(nil)) {
		t.Fatal("clone state is not independent")
	}
}
