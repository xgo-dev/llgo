package build

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCommandEnvConfigureEnforcesErrDot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX shell scripts")
	}
	workDir := t.TempDir()
	relBin := filepath.Join(workDir, "tool-bin")
	if err := os.MkdirAll(relBin, 0o755); err != nil {
		t.Fatal(err)
	}
	toolPath := filepath.Join(relBin, "fake-tool")
	if err := os.WriteFile(toolPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GODEBUG", "")
	env := commandEnv{dir: workDir, environ: []string{"PATH=tool-bin"}}
	cmd := env.configure(exec.Command("fake-tool"))
	if !errors.Is(cmd.Err, exec.ErrDot) {
		t.Fatalf("configure(relative PATH) cmd.Err = %v, want exec.ErrDot", cmd.Err)
	}
	if _, err := env.lookPath("fake-tool"); !errors.Is(err, exec.ErrDot) {
		t.Fatalf("lookPath(relative PATH) = %v, want exec.ErrDot", err)
	}

	envAllow := commandEnv{dir: workDir, environ: []string{"PATH=tool-bin", "GODEBUG=execerrdot=0"}}
	cmdAllow := envAllow.configure(exec.Command("fake-tool"))
	if cmdAllow.Err != nil {
		t.Fatalf("configure(relative PATH, execerrdot=0) cmd.Err = %v, want nil", cmdAllow.Err)
	}
	if filepath.Clean(cmdAllow.Path) != toolPath {
		t.Fatalf("configure(relative PATH, execerrdot=0) cmd.Path = %q, want %q", cmdAllow.Path, toolPath)
	}
}
