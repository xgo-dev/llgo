//go:build go1.25

package os_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRootFileOperations(t *testing.T) {
	directory := t.TempDir()
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if err := root.MkdirAll("nested/dir", 0755); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile("nested/dir/source.txt", []byte("contents"), 0644); err != nil {
		t.Fatal(err)
	}
	data, err := root.ReadFile("nested/dir/source.txt")
	if err != nil || string(data) != "contents" {
		t.Fatalf("ReadFile = %q, %v; want contents, nil", data, err)
	}
	if err := root.Chmod("nested/dir/source.txt", 0600); err != nil {
		t.Fatal(err)
	}
	when := time.Unix(123456789, 0)
	if err := root.Chtimes("nested/dir/source.txt", when, when); err != nil {
		t.Fatal(err)
	}
	if err := root.Chown("nested/dir/source.txt", -1, -1); err != nil {
		t.Fatal(err)
	}
	if err := root.Link("nested/dir/source.txt", "nested/hardlink.txt"); err != nil {
		t.Fatal(err)
	}
	if err := root.Rename("nested/hardlink.txt", "nested/renamed.txt"); err != nil {
		t.Fatal(err)
	}
	if err := root.Symlink("dir/source.txt", "nested/symlink.txt"); err != nil {
		t.Fatal(err)
	}
	if err := root.Lchown("nested/symlink.txt", -1, -1); err != nil {
		t.Fatal(err)
	}
	target, err := root.Readlink("nested/symlink.txt")
	if err != nil || target != "dir/source.txt" {
		t.Fatalf("Readlink = %q, %v; want dir/source.txt, nil", target, err)
	}
	if err := root.RemoveAll("nested"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(directory, "nested")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("RemoveAll left nested directory: %v", err)
	}
}
