package ioutil_test

import (
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestReadDir(t *testing.T) {
	dir := t.TempDir()
	names := []string{"alpha.txt", "beta.txt", "gopher"}
	for _, name := range names[:2] {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
			t.Fatalf("WriteFile %s error: %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, names[2]), 0o755); err != nil {
		t.Fatalf("Mkdir %s error: %v", names[2], err)
	}

	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir error: %v", err)
	}
	if len(infos) != len(names) {
		t.Fatalf("ReadDir returned %d entries, want %d", len(infos), len(names))
	}

	got := make(map[string]fs.FileInfo)
	for _, info := range infos {
		got[info.Name()] = info
	}
	for _, name := range names {
		if _, ok := got[name]; !ok {
			t.Fatalf("ReadDir missing entry %s", name)
		}
	}
	if !got[names[2]].IsDir() {
		t.Fatalf("ReadDir expected %s to be directory", names[2])
	}
}
