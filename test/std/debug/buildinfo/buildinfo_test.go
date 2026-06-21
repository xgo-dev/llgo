//go:build !llgo

package buildinfo_test

import (
	"debug/buildinfo"
	"os"
	"testing"
)

func TestReadFileAndType(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable failed: %v", err)
	}

	info, err := buildinfo.ReadFile(exe)
	if err == nil {
		if info == nil {
			t.Fatalf("ReadFile returned nil info with nil error")
		}
		if info.GoVersion == "" {
			t.Fatalf("ReadFile returned empty GoVersion")
		}
	}

	// Ensure the exported alias type is referenced by tests.
	var _ *buildinfo.BuildInfo = info
}

func TestRead(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable failed: %v", err)
	}
	f, err := os.Open(exe)
	if err != nil {
		t.Fatalf("Open(%q) failed: %v", exe, err)
	}
	defer f.Close()

	info, err := buildinfo.Read(f)
	if err == nil {
		if info == nil {
			t.Fatalf("Read returned nil info with nil error")
		}
		if info.GoVersion == "" {
			t.Fatalf("Read returned empty GoVersion")
		}
	}
}
