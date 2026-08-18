//go:build go1.26

package token_test

import (
	"go/token"
	"testing"
)

func TestFileEnd(t *testing.T) {
	var source token.FileSet
	file := source.AddFile("source.go", -1, 10)
	if got, want := file.End(), token.Pos(file.Base()+file.Size()); got != want {
		t.Fatalf("File.End = %d, want %d", got, want)
	}
}
