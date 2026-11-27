//go:build js && wasm

package wasmtest

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// Exercise the unmodified Go syscall/fs_js.go callback path. A working host
// callback bridge must not require replacing ordinary file I/O with *Sync.
func TestJSCallFileIO(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "数据.txt")
	const initial = "hello\x00wasm"
	if err := os.WriteFile(name, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if n, err := f.WriteAt([]byte("GO"), 6); err != nil || n != 2 {
		t.Fatalf("WriteAt = %d, %v", n, err)
	}
	if offset, err := f.Seek(6, io.SeekStart); err != nil || offset != 6 {
		t.Fatalf("Seek = %d, %v", offset, err)
	}
	var tail [4]byte
	if n, err := io.ReadFull(f, tail[:]); err != nil || n != len(tail) || string(tail[:]) != "GOsm" {
		t.Fatalf("ReadFull = %q, %d, %v", tail, n, err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	renamed := filepath.Join(dir, "renamed.txt")
	if err := os.Rename(name, renamed); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(renamed)
	if err != nil || string(got) != "hello\x00GOsm" {
		t.Fatalf("ReadFile = %q, %v", got, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "renamed.txt" {
		t.Fatalf("ReadDir = %v, %v", entries, err)
	}
	info, err := os.Stat(renamed)
	if err != nil || info.Size() != int64(len(initial)) {
		t.Fatalf("Stat = %v, %v", info, err)
	}
}

func TestJSCallFileErrors(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")
	_, err := os.ReadFile(missing)
	var pathErr *fs.PathError
	if !errors.Is(err, fs.ErrNotExist) || !errors.As(err, &pathErr) || pathErr.Path != missing {
		t.Fatalf("missing file error = %v", err)
	}
	if err := os.WriteFile(missing, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(missing, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if f != nil {
		f.Close()
	}
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("exclusive create error = %v", err)
	}
}
