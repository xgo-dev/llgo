package cltest

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/xgo-dev/llgo/internal/littest"
)

func TestFilterRunOutputToolchainWarnings(t *testing.T) {
	in := []byte("ld64.lld: warning: library is newer than target minimum\n" +
		"'+zcm' is not a recognized feature for this target (ignoring feature)\n" +
		"'+zcz' is not a recognized feature for this target (ignoring feature)\n" +
		"pass\n")
	want := []byte("pass\n")
	if got := filterRunOutput(in); !bytes.Equal(got, want) {
		t.Fatalf("filterRunOutput() = %q, want %q", got, want)
	}
}

func TestAdditionalIRTargets(t *testing.T) {
	targets := []littest.Target{
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "amd64"},
	}
	tests := []struct {
		name          string
		currentPrefix string
		want          []littest.Target
	}{
		{name: "listed current", currentPrefix: "DARWIN-ARM64", want: targets[1:]},
		{name: "unlisted current", currentPrefix: "LINUX-ARM64", want: targets},
		{name: "named current", currentPrefix: "TARGET-WASM", want: targets},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := additionalIRTargets(targets, test.currentPrefix); !slices.Equal(got, test.want) {
				t.Fatalf("additionalIRTargets() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestReadGoldenUsesToolchainVersion(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "expect.txt")
	if err := os.WriteFile(file, []byte("default\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	versioned, ok := goldenForGoVersion(file, runtime.Version())
	if !ok {
		t.Fatalf("runtime version %q is not a valid Go version", runtime.Version())
	}
	if err := os.WriteFile(versioned, []byte("versioned\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, check, err := readGolden(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "versioned\n" || !check {
		t.Fatalf("readGolden() = %q, %v, want versioned golden", got, check)
	}

	if err := os.Remove(versioned); err != nil {
		t.Fatal(err)
	}
	got, check, err = readGolden(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "default\n" || !check {
		t.Fatalf("readGolden() fallback = %q, %v, want default golden", got, check)
	}
}

func TestGoldenForGoVersionRejectsUnversionedToolchain(t *testing.T) {
	if got, ok := goldenForGoVersion("expect.txt", "devel custom"); ok || got != "" {
		t.Fatalf("goldenForGoVersion() = %q, %v, want no versioned golden", got, ok)
	}
}

func TestReadGoldenReturnsVersionedReadError(t *testing.T) {
	file := filepath.Join(t.TempDir(), "expect.txt")
	versioned, ok := goldenForGoVersion(file, runtime.Version())
	if !ok {
		t.Fatalf("runtime version %q is not a valid Go version", runtime.Version())
	}
	if err := os.Mkdir(versioned, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, _, err := readGolden(file); err == nil {
		t.Fatal("readGolden() succeeded for a versioned golden directory")
	}
}
