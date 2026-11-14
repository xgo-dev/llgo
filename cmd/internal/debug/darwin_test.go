//go:build !llgo

package debug

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDarwinLTOSymbolsRejectsMissingDebugObjects(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin dSYM sessions")
	}
	directory := t.TempDir()
	artifact := filepath.Join(directory, "program with spaces")
	if args, cleanup, err := darwinLTOSymbols(artifact); err != nil || len(args) != 0 {
		t.Fatalf("ordinary artifact prepared LTO symbols: %v, %v", args, err)
	} else {
		cleanup()
	}
	if err := os.Mkdir(artifact+".llgo-debug-lto", 0o700); err != nil {
		t.Fatal(err)
	}
	// dsymutil can return success after warning about unavailable objects.
	// Never attach an empty/missing symbol file in that case.
	if err := os.WriteFile(filepath.Join(directory, "dsymutil"), []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TMPDIR", directory)
	args, cleanup, err := darwinLTOSymbols(artifact)
	defer cleanup()
	if err == nil || !strings.Contains(err.Error(), "read Darwin LTO symbols") || len(args) != 0 {
		t.Fatalf("unavailable symbols were accepted: %v, %v", args, err)
	}
	if dirs, err := filepath.Glob(filepath.Join(directory, "llgo-debug-dsym-*")); err != nil || len(dirs) != 0 {
		t.Fatalf("failed preparation leaked private symbols: %v, %v", dirs, err)
	}
}
