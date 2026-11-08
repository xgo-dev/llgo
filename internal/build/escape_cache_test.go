package build

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/goescape"
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

func TestEscapeCacheShardLimitPreservesPublishedFacts(t *testing.T) {
	dir := t.TempDir()
	newFiles := func() *escapeCacheFiles {
		return &escapeCacheFiles{dir: dir, shards: make(map[string]*escapeCacheShard)}
	}
	bucket := strings.Repeat("a", 64)
	oldKey, newKey := bucket+"."+strings.Repeat("b", 64), bucket+"."+strings.Repeat("c", 64)
	// Either record fits, but their combined JSON exceeds the shard limit.
	data := bytes.Repeat([]byte{'x'}, goescape.MaxCacheBytes/2+2)
	data[0], data[len(data)-1] = '"', '"'
	old := newFiles()
	if err := old.store(oldKey, data); err != nil {
		t.Fatal(err)
	}
	old.flush()

	current := newFiles()
	if err := current.store(newKey, data); err != nil {
		t.Fatal(err)
	}
	current.flush() // Evict the previous build's unused record.
	checkPublished := func() {
		t.Helper()
		fresh := newFiles()
		if got, err := fresh.load(newKey); err != nil || !bytes.Equal(got, data) {
			t.Fatalf("current facts were not preserved: %v", err)
		}
		if _, err := fresh.load(oldKey); !os.IsNotExist(err) {
			t.Fatalf("unused or unpublished facts were retained: %v", err)
		}
	}
	checkPublished()

	// Both records are now used by this build. An oversized active set must
	// leave the last published shard intact, so a later build can still use it.
	if err := current.store(oldKey, data); err != nil {
		t.Fatal(err)
	}
	current.flush()
	checkPublished()
}
