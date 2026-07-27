//go:build !llgo

package llvm

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNewWithEnvUsesExplicitToolchainEnvironment(t *testing.T) {
	binDir := t.TempDir()
	llvmConfig := filepath.Join(binDir, "llvm-config")
	readelf := filepath.Join(binDir, "llvm-readelf")
	writeExecutable(t, llvmConfig, "#!/bin/sh\nprintf '%s\\n' \""+binDir+"\"\n")
	writeExecutable(t, readelf, "#!/bin/sh\ntest \"$REQUEST_MARKER\" = expected\n")

	environ := []string{
		"PATH=" + binDir,
		"LLVM_CONFIG=" + llvmConfig,
		"REQUEST_MARKER=expected",
	}
	env := NewWithEnv("", environ)
	if got := env.BinDir(); got != binDir {
		t.Fatalf("BinDir() = %q, want %q", got, binDir)
	}
	cmd, err := env.Readelf("--version")
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(); err != nil {
		t.Fatalf("Readelf did not use explicit environment: %v", err)
	}
}

func TestNewWithEnvRejectsRelativePathFromWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.Mkdir(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	llvmConfig := filepath.Join(binDir, "llvm-config")
	readelf := filepath.Join(binDir, "llvm-readelf")
	writeExecutable(t, llvmConfig, "#!/bin/sh\nprintf 'bin\\n'\n")
	writeExecutable(t, readelf, "#!/bin/sh\nexit 0\n")

	env := NewWithEnv("", []string{"PATH=bin", "LLVM_CONFIG=/bin/false"}, dir)
	cmd, err := env.Readelf("--version")
	if !errors.Is(err, exec.ErrDot) {
		t.Fatalf("Readelf error = %v, want exec.ErrDot", err)
	}
	if cmd != nil {
		t.Fatalf("Readelf command = %v, want nil", cmd)
	}
}

func writeExecutable(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
}
