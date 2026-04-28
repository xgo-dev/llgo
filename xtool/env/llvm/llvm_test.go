package llvm

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestNewCachesSuccessfulBindir(t *testing.T) {
	if os.PathSeparator != '/' {
		t.Skip("fake llvm-config shell script is Unix-only")
	}
	bindirCache = sync.Map{}
	dir := t.TempDir()
	countFile := filepath.Join(dir, "count")
	config := filepath.Join(dir, "llvm-config")
	script := "#!/bin/sh\ncount=0\nif [ -f " + countFile + " ]; then count=$(cat " + countFile + "); fi\ncount=$((count + 1))\necho $count > " + countFile + "\necho /tmp/llvm-bin\n"
	if err := os.WriteFile(config, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	if got := New(config).BinDir(); got != "/tmp/llvm-bin" {
		t.Fatalf("first BinDir = %q", got)
	}
	if got := New(config).BinDir(); got != "/tmp/llvm-bin" {
		t.Fatalf("second BinDir = %q", got)
	}
	count, err := os.ReadFile(countFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(count)) != "1" {
		t.Fatalf("llvm-config invoked %s times, want 1", strings.TrimSpace(string(count)))
	}
}

func TestNewDoesNotCacheBindirFailures(t *testing.T) {
	if os.PathSeparator != '/' {
		t.Skip("fake llvm-config shell script is Unix-only")
	}
	bindirCache = sync.Map{}
	dir := t.TempDir()
	countFile := filepath.Join(dir, "count")
	config := filepath.Join(dir, "llvm-config")
	script := "#!/bin/sh\ncount=0\nif [ -f " + countFile + " ]; then count=$(cat " + countFile + "); fi\ncount=$((count + 1))\necho $count > " + countFile + "\nif [ $count -eq 1 ]; then exit 2; fi\necho /tmp/llvm-bin\n"
	if err := os.WriteFile(config, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	if got := New(config).BinDir(); got != "" {
		t.Fatalf("failed BinDir = %q, want empty", got)
	}
	if got := New(config).BinDir(); got != "/tmp/llvm-bin" {
		t.Fatalf("retry BinDir = %q", got)
	}
}
