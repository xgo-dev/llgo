//go:build !llgo && unix

package processenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookPathSkipsFileNotExecutableByCurrentUser(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root execute-access semantics differ")
	}
	firstDir := t.TempDir()
	secondDir := t.TempDir()
	firstTool := filepath.Join(firstDir, "snapshot-tool")
	if err := os.WriteFile(firstTool, []byte("#!/bin/sh\nexit 1\n"), 0o001); err != nil {
		t.Fatal(err)
	}
	secondTool := filepath.Join(secondDir, "snapshot-tool")
	if err := os.WriteFile(secondTool, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	pathEnv := firstDir + string(os.PathListSeparator) + secondDir
	path, err := LookPath([]string{"PATH=" + pathEnv}, "", "snapshot-tool")
	if err != nil {
		t.Fatal(err)
	}
	if path != secondTool {
		t.Fatalf("LookPath path = %q, want %q", path, secondTool)
	}
}
