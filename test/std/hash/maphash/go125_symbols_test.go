//go:build go1.25

package maphash_test

import (
	"bytes"
	"hash/maphash"
	"testing"
)

func TestClone(t *testing.T) {
	var original maphash.Hash
	original.SetSeed(maphash.MakeSeed())
	original.WriteString("prefix:")
	clone, err := original.Clone()
	if err != nil {
		t.Fatal(err)
	}
	original.WriteString("same")
	clone.Write([]byte("same"))
	if !bytes.Equal(original.Sum(nil), clone.Sum(nil)) {
		t.Fatal("clone did not preserve hash state")
	}
	clone.Write([]byte(":different"))
	if bytes.Equal(original.Sum(nil), clone.Sum(nil)) {
		t.Fatal("clone state is not independent")
	}
}
