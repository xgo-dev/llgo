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

func TestCaptureClonesExplicitInputs(t *testing.T) {
	workDir := t.TempDir()
	environ := []string{"KEY=value"}
	process, err := Capture(workDir, environ)
	if err != nil {
		t.Fatal(err)
	}
	environ[0] = "KEY=changed"
	if got := process.Get("KEY"); got != "value" {
		t.Fatalf("Get(KEY) = %q, want value", got)
	}
	if got := process.Abs("out.o"); got != filepath.Join(workDir, "out.o") {
		t.Fatalf("Abs(out.o) = %q", got)
	}
	if got := process.Abs(""); got != "" {
		t.Fatalf("Abs(empty) = %q", got)
	}
	if got := process.Abs(workDir); got != workDir {
		t.Fatalf("Abs(absolute) = %q", got)
	}
	if got, ok := process.Lookup("KEY"); !ok || got != "value" {
		t.Fatalf("Lookup(KEY) = %q, %v, want value, true", got, ok)
	}

	clone := process.Clone()
	clone.Env[0] = "KEY=clone"
	if got := process.Get("KEY"); got != "value" {
		t.Fatalf("clone changed source context: Get(KEY) = %q", got)
	}
}

func TestCaptureSnapshotsAmbientInputs(t *testing.T) {
	t.Setenv("PROCESSENV_CAPTURE_TEST", "before")
	process, err := Capture("", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PROCESSENV_CAPTURE_TEST", "after")
	if got := process.Get("PROCESSENV_CAPTURE_TEST"); got != "before" {
		t.Fatalf("Get(PROCESSENV_CAPTURE_TEST) = %q, want before", got)
	}
	if process.Dir == "" || !filepath.IsAbs(process.Dir) {
		t.Fatalf("Capture directory = %q, want absolute working directory", process.Dir)
	}
}

func TestZeroContextUsesAmbientEnvironment(t *testing.T) {
	t.Setenv("PROCESSENV_CONTEXT_TEST", "ambient")
	var process Context
	if got := process.Get("PROCESSENV_CONTEXT_TEST"); got != "ambient" {
		t.Fatalf("Get(PROCESSENV_CONTEXT_TEST) = %q, want ambient", got)
	}
	if got, ok := process.Lookup("PROCESSENV_CONTEXT_TEST"); !ok || got != "ambient" {
		t.Fatalf("Lookup(PROCESSENV_CONTEXT_TEST) = %q, %v, want ambient, true", got, ok)
	}
}

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

	process := Context{
		Dir: workDir,
		Env: []string{
			"PATH=" + binDir,
			"REQUEST_VALUE=snapshot",
		},
	}
	cmd := process.Command("snapshot-tool")
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

func TestNilEnvironmentUsesProcessEnvironment(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := Command(nil, "", executable)
	if cmd.Env != nil {
		t.Fatalf("Command environment = %v, want inherited environment", cmd.Env)
	}
	if cmd.Path != executable {
		t.Fatalf("Command path = %q, want %q", cmd.Path, executable)
	}

	path, err := LookPath(nil, "", executable)
	if err != nil {
		t.Fatal(err)
	}
	if path != executable {
		t.Fatalf("LookPath path = %q, want %q", path, executable)
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
