package build

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEscapeCacheBatchesAndRepairsShards(t *testing.T) {
	dir := t.TempDir()
	newFiles := func() *escapeCacheFiles {
		return &escapeCacheFiles{dir: dir, shards: make(map[string]*escapeCacheShard)}
	}
	files := newFiles()
	bucket := strings.Repeat("a", 64)
	keys := []string{bucket + "." + strings.Repeat("b", 64), bucket + "." + strings.Repeat("c", 64)}
	data := []byte(`{"record":"complete"}`)
	for _, key := range keys {
		if err := files.store(key, data); err != nil {
			t.Fatal(err)
		}
	}
	files.flush()
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(paths) != 1 {
		t.Fatalf("expected one package shard: %v %v", paths, err)
	}
	fresh := newFiles()
	for _, key := range keys {
		got, err := fresh.load(key)
		if err != nil || !bytes.Equal(data, got) {
			t.Fatalf("persisted record=%s %v", got, err)
		}
	}
	if err := os.WriteFile(paths[0], []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	fresh = newFiles()
	if _, err := fresh.load(keys[0]); !os.IsNotExist(err) {
		t.Fatalf("corrupt shard did not miss: %v", err)
	}
	if err := fresh.store(keys[0], data); err != nil {
		t.Fatal(err)
	}
	fresh.flush()
	if got, err := newFiles().load(keys[0]); err != nil || !bytes.Equal(got, data) {
		t.Fatal("shard was not repaired", err)
	}
	if _, err := fresh.load("../../other"); err == nil {
		t.Fatal("invalid key accepted")
	}
}
