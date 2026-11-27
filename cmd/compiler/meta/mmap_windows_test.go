//go:build windows

package meta

import (
	"os"
	"testing"
)

func TestMapFileErrors(t *testing.T) {
	f, err := os.CreateTemp("", "llgo-meta-mmap-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	})
	if _, err := f.Write([]byte{0}); err != nil {
		t.Fatal(err)
	}

	if raw, err := mapFile(f, int(^uint(0)>>1)); err == nil {
		_ = unmapFile(raw)
		t.Fatal("mapFile unexpectedly mapped a view larger than its file mapping")
	}

	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := mapFile(f, 1); err == nil {
		t.Fatal("mapFile unexpectedly accepted a closed file")
	}
	if err := unmapFile(nil); err != nil {
		t.Fatalf("unmapFile(nil) = %v, want nil", err)
	}
}
