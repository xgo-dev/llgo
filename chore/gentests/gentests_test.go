package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirHasLITTESTSource(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "in.go"), []byte("// LITTEST\npackage main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ok, err := dirHasLITTESTSource(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("dirHasLITTESTSource = false, want true")
	}
}

func TestDirHasLITTESTSource_IgnoresUnmarkedFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "in.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ok, err := dirHasLITTESTSource(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("dirHasLITTESTSource = true, want false")
	}
}
