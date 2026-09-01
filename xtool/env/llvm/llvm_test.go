//go:build !llgo

package llvm

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestParseMajorVersion(t *testing.T) {
	tests := []struct {
		version string
		want    int
	}{
		{version: "21.1.8", want: 21},
		{version: "Homebrew clang version 21.1.8", want: 21},
		{version: "Homebrew LLD 21.1.8 (compatible with GNU linkers)", want: 21},
	}
	for _, test := range tests {
		got, err := parseMajorVersion(test.version)
		if err != nil {
			t.Fatalf("parseMajorVersion(%q): %v", test.version, err)
		}
		if got != test.want {
			t.Fatalf("parseMajorVersion(%q) = %d, want %d", test.version, got, test.want)
		}
	}
}

func TestValidateToolchainMajor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a shell script")
	}

	matching := writeVersionTool(t, "clang 21.1.3")
	if err := ValidateToolchainMajor("21.1.8", matching); err != nil {
		t.Fatalf("matching major rejected: %v", err)
	}

	mismatched := writeVersionTool(t, "clang 19.1.7")
	err := ValidateToolchainMajor("21.1.8", mismatched)
	if err == nil || !strings.Contains(err.Error(), "LLVM major version mismatch") {
		t.Fatalf("mismatched major error = %v", err)
	}
}

func writeVersionTool(t *testing.T, version string) string {
	t.Helper()
	tool := filepath.Join(t.TempDir(), "llvm-tool")
	contents := "#!/bin/sh\nprintf '%s\\n' '" + version + "'\n"
	if err := os.WriteFile(tool, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
	return tool
}
