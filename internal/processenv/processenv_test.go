//go:build !llgo

package processenv

import (
	"errors"
	"os"
	"os/exec"
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
		"PATH=" + binDir,
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

func TestLookPathRejectsRelativePathEntries(t *testing.T) {
	workDir := t.TempDir()
	binDir := filepath.Join(workDir, "bin")
	if err := os.Mkdir(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(binDir, "snapshot-tool")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	path, err := LookPath([]string{"PATH=bin"}, workDir, "snapshot-tool")
	if !errors.Is(err, exec.ErrDot) {
		t.Fatalf("LookPath error = %v, want exec.ErrDot", err)
	}
	if path != tool {
		t.Fatalf("LookPath path = %q, want %q", path, tool)
	}
	cmd := Command([]string{"PATH=bin"}, workDir, "snapshot-tool")
	if err := cmd.Run(); !errors.Is(err, exec.ErrDot) {
		t.Fatalf("Command error = %v, want exec.ErrDot", err)
	}

	rootTool := filepath.Join(workDir, "root-tool")
	if err := os.WriteFile(rootTool, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	path, err = LookPath([]string{"PATH=" + string(os.PathListSeparator)}, workDir, "root-tool")
	if !errors.Is(err, exec.ErrDot) || path != rootTool {
		t.Fatalf("empty PATH entry resolved to %q, %v; want %q, exec.ErrDot", path, err, rootTool)
	}
}

func TestLookPathAllowsExplicitRelativeName(t *testing.T) {
	workDir := t.TempDir()
	binDir := filepath.Join(workDir, "bin")
	if err := os.Mkdir(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(binDir, "snapshot-tool")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := Command([]string{}, workDir, filepath.Join("bin", "snapshot-tool"))
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
}

func TestLookupAndMissingExecutable(t *testing.T) {
	environ := []string{"KEY=first", "EMPTY=", "KEY=last"}
	if got, ok := Lookup(environ, "KEY"); !ok || got != "last" {
		t.Fatalf("Lookup(KEY) = %q, %v, want last, true", got, ok)
	}
	if got := Get(environ, "EMPTY"); got != "" {
		t.Fatalf("Get(EMPTY) = %q, want empty", got)
	}
	if _, ok := Lookup(environ, "MISSING"); ok {
		t.Fatal("Lookup(MISSING) reported present")
	}
	if _, err := LookPath([]string{"PATH=" + t.TempDir()}, "", "missing-tool"); !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("LookPath missing error = %v, want exec.ErrNotFound", err)
	}

	nonExecutable := filepath.Join(t.TempDir(), "non-executable")
	if err := os.WriteFile(nonExecutable, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LookPath([]string{}, "", nonExecutable); !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("LookPath non-executable error = %v, want exec.ErrNotFound", err)
	}
	if _, err := LookPath([]string{}, "", t.TempDir()); !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("LookPath directory error = %v, want exec.ErrNotFound", err)
	}
}
