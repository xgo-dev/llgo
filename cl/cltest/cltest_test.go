package cltest

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goplus/llgo/internal/build"
)

func TestWithModuleCaptureDWARFMode(t *testing.T) {
	for _, test := range []struct {
		name string
		mode build.DWARFMode
		want build.DWARFMode
	}{
		{name: "default", mode: build.DWARFDefault, want: build.DWARFOmit},
		{name: "preserve", mode: build.DWARFPreserve, want: build.DWARFPreserve},
		{name: "omit", mode: build.DWARFOmit, want: build.DWARFOmit},
	} {
		t.Run(test.name, func(t *testing.T) {
			conf := build.NewDefaultConf(build.ModeRun)
			conf.LinkOptions.DWARF = test.mode
			got, _, _ := withModuleCapture(conf, t.TempDir())
			if got.LinkOptions.DWARF != test.want {
				t.Fatalf("DWARF mode = %v, want %v", got.LinkOptions.DWARF, test.want)
			}
			if conf.LinkOptions.DWARF != test.mode {
				t.Fatalf("input DWARF mode = %v, want %v", conf.LinkOptions.DWARF, test.mode)
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
