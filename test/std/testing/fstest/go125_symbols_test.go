//go:build go1.25

package fstest_test

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestMapFSSymlink(t *testing.T) {
	filesystem := fstest.MapFS{
		"target": &fstest.MapFile{Data: []byte("contents")},
		"link":   &fstest.MapFile{Data: []byte("target"), Mode: fs.ModeSymlink},
	}
	info, err := filesystem.Lstat("link")
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("Lstat mode = %v, want symlink", info.Mode())
	}
	target, err := filesystem.ReadLink("link")
	if err != nil || target != "target" {
		t.Fatalf("ReadLink = %q, %v; want target, nil", target, err)
	}
}
