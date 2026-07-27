//go:build !llgo

package processenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandUsesSnapshotPathEnvironmentAndDir(t *testing.T) {
	workDir := t.TempDir()
	binDir := filepath.Join(workDir, "bin")
	if err := os.Mkdir(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(binDir, "snapshot-tool")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nprintf '%s:%s' \"$REQUEST_VALUE\" \"$PWD\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := Command([]string{
		"PATH=bin",
		"REQUEST_VALUE=snapshot",
	}, workDir, "snapshot-tool")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	resolvedWorkDir, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(out), "snapshot:"+resolvedWorkDir; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
	if !strings.HasSuffix(cmd.Path, filepath.Join("bin", "snapshot-tool")) {
		t.Fatalf("command path = %q", cmd.Path)
	}
}
