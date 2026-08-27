package llvm

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSetupPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a shell script")
	}

	binDir := t.TempDir()
	llvmConfig := filepath.Join(t.TempDir(), "llvm-config")
	if err := os.WriteFile(llvmConfig, []byte("#!/bin/sh\nprintf '%s\\n' \"${LLGO_TEST_LLVM_BINDIR}\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLVM_CONFIG", llvmConfig)
	t.Setenv("LLGO_TEST_LLVM_BINDIR", binDir)
	original := filepath.Join(t.TempDir(), "original")
	t.Setenv("PATH", original)

	SetupPath()
	want := binDir + string(os.PathListSeparator) + original
	if got := os.Getenv("PATH"); got != want {
		t.Fatalf("PATH = %q, want %q", got, want)
	}

	SetupPath()
	if got := os.Getenv("PATH"); got != want {
		t.Fatalf("second setup changed PATH to %q, want %q", got, want)
	}
}

func TestSetupPathIgnoresMissingBinDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a shell script")
	}

	llvmConfig := filepath.Join(t.TempDir(), "llvm-config")
	if err := os.WriteFile(llvmConfig, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLVM_CONFIG", llvmConfig)
	t.Setenv("PATH", filepath.Join(t.TempDir(), "original"))
	before := os.Getenv("PATH")

	SetupPath()
	if got := os.Getenv("PATH"); got != before {
		t.Fatalf("PATH changed from %q to %q without an LLVM bin directory", before, got)
	}
}
