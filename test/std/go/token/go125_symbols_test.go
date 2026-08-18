//go:build go1.25

package token_test

import (
	"go/token"
	"testing"
)

func TestAddExistingFiles(t *testing.T) {
	var source token.FileSet
	file := source.AddFile("source.go", -1, 10)

	var destination token.FileSet
	destination.AddExistingFiles(file)
	if got := destination.File(file.Pos(5)); got != file {
		t.Fatalf("FileSet.File returned %p, want %p", got, file)
	}
	count := 0
	destination.Iterate(func(got *token.File) bool {
		count++
		return got == file
	})
	if count != 1 {
		t.Fatalf("FileSet contains %d files, want 1", count)
	}
}
