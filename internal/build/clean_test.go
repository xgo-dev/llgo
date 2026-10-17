package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
)

func TestCleanMainPkgRemovesSidecars(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	sourceDir := filepath.Join(root, "source")
	for _, dir := range []string{binDir, sourceDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	pkg := &packages.Package{
		PkgPath:         "example.com/tools/demo",
		CompiledGoFiles: []string{filepath.Join(sourceDir, "main.go")},
	}
	conf := &Config{BinPath: binDir, AppExt: ".exe"}
	outputs := []string{
		filepath.Join(binDir, "demo.exe"),
		filepath.Join(sourceDir, "demo.exe"),
	}
	for _, executable := range outputs {
		if err := os.MkdirAll(filepath.Join(executable+".dSYM", "Contents", "Resources", "DWARF"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(executable+".dSYM", "Contents", "Info.plist"), []byte("metadata"), 0o644); err != nil {
			t.Fatal(err)
		}
		for _, artifact := range []string{executable, pclnSidecarPath(executable), dwarfSidecarPath(executable)} {
			if err := os.WriteFile(artifact, []byte("artifact"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	keep := filepath.Join(sourceDir, "demo.exe.pclntab.keep")
	if err := os.WriteFile(keep, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	cleanMainPkg(pkg, conf, false)

	for _, executable := range outputs {
		for _, artifact := range []string{executable, pclnSidecarPath(executable), dwarfSidecarPath(executable), executable + ".dSYM"} {
			if _, err := os.Stat(artifact); !os.IsNotExist(err) {
				t.Errorf("artifact %q still exists (stat error %v)", artifact, err)
			}
		}
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("unrelated file was removed: %v", err)
	}
}

func TestCleanPackageExportsAndListErrors(t *testing.T) {
	dir := t.TempDir()
	export := filepath.Join(dir, "package.a")
	write := func(file string) {
		t.Helper()
		if err := os.WriteFile(file, []byte("artifact"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(export + ".ll")
	if err := cleanPkgs([]*packages.Package{{PkgPath: "example.com/p", ExportFile: export}, {PkgPath: "synthetic"}}, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(export + ".ll"); !os.IsNotExist(err) {
		t.Fatal("did not remove the LLGo export artifact")
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/clean\ngo 1.24\n"), 0600); err != nil {
		t.Fatal(err)
	}
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)
	t.Setenv("GOWORK", "off")
	t.Setenv("GOENV", "off")
	t.Setenv("GOFLAGS", "")
	if err := Clean([]string{filepath.Join(dir, "missing")}, &Config{}); err == nil {
		t.Fatal("silently ignored invalid package")
	}
	// Successful cleanup also preserves gc's export file.
	if err := os.WriteFile(filepath.Join(dir, "value.go"), []byte("package p\nconst Value=42\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Clean([]string{"."}, &Config{}); err != nil {
		t.Fatal(err)
	}
	// The command layer ignores build-only LLGo defaults for clean. The Go
	// package loader must not try to parse those defaults a second time.
	t.Setenv("GOFLAGS", "-size")
	if err := Clean([]string{"."}, &Config{GoFlagsParsed: true}); err != nil {
		t.Fatalf("reparsed native command defaults: %v", err)
	}
	if got := os.Getenv("GOFLAGS"); got != "-size" {
		t.Fatalf("clean changed caller GOFLAGS: %q", got)
	}
}

func TestCleanMainAndDriverFailures(t *testing.T) {
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	t.Setenv("GOENV", "off")
	t.Setenv("GOFLAGS", "")
	t.Setenv("GOWORK", "off")
	if err := os.WriteFile("go.mod", []byte("module example.com/program\ngo 1.24\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("main.go", []byte("package main\nfunc main(){}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("program", []byte("artifact"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Clean(nil, &Config{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("program"); !os.IsNotExist(err) {
		t.Fatal("did not remove main artifact")
	}
	t.Setenv("GOPACKAGESDRIVER", filepath.Join(dir, "missing-driver"))
	if err := Clean(nil, &Config{}); err == nil {
		t.Fatal("did not report driver failure")
	}
}
