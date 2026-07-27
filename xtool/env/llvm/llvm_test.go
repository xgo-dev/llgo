//go:build !llgo

package llvm

import (
	"os"
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

func TestNewWithEnvResolvesRelativePathFromWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.Mkdir(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	llvmConfig := filepath.Join(binDir, "llvm-config")
	readelf := filepath.Join(binDir, "llvm-readelf")
	writeExecutable(t, llvmConfig, "#!/bin/sh\nprintf 'bin\\n'\n")
	writeExecutable(t, readelf, "#!/bin/sh\nexit 0\n")

	env := NewWithEnv("", []string{"PATH=bin"}, dir)
	cmd, err := env.Readelf("--version")
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(); err != nil {
		t.Fatalf("Readelf did not resolve relative PATH from request directory: %v", err)
	}
}

func writeExecutable(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
}
