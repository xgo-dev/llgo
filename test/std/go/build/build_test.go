package build_test

import (
	"errors"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestImportDirBasic(t *testing.T) {
	dir := t.TempDir()

	source := "package gobuildtest\nimport \"fmt\"\n\nfunc Hello() string { return fmt.Sprint(\"hi\") }\n"
	if err := os.WriteFile(filepath.Join(dir, "gobuild.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	ctx := build.Default
	pkg, err := ctx.ImportDir(dir, 0)
	if err != nil {
		t.Fatalf("ImportDir returned error: %v", err)
	}

	if pkg.Name != "gobuildtest" {
		t.Fatalf("ImportDir package name = %q, want %q", pkg.Name, "gobuildtest")
	}

	if len(pkg.GoFiles) != 1 || pkg.GoFiles[0] != "gobuild.go" {
		t.Fatalf("ImportDir GoFiles = %v, want [gobuild.go]", pkg.GoFiles)
	}

	foundFmt := false
	for _, imp := range pkg.Imports {
		if imp == "fmt" {
			foundFmt = true
			break
		}
	}
	if !foundFmt {
		t.Fatalf("ImportDir imports %v, want to include fmt", pkg.Imports)
	}
}

func TestMatchFileWithBuildTags(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "positive.go"), []byte("//go:build mytag\n\npackage sample\n"), 0o644); err != nil {
		t.Fatalf("write positive file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "negative.go"), []byte("//go:build !mytag\n\npackage sample\n"), 0o644); err != nil {
		t.Fatalf("write negative file: %v", err)
	}

	ctx := build.Default
	ctx.BuildTags = append([]string{"mytag"}, ctx.BuildTags...)

	matched, err := ctx.MatchFile(dir, "positive.go")
	if err != nil {
		t.Fatalf("MatchFile positive.go returned error: %v", err)
	}
	if !matched {
		t.Fatal("MatchFile positive.go = false, want true")
	}

	matched, err = ctx.MatchFile(dir, "negative.go")
	if err != nil {
		t.Fatalf("MatchFile negative.go returned error: %v", err)
	}
	if matched {
		t.Fatal("MatchFile negative.go = true, want false")
	}
}

func TestImportDirNoGoError(t *testing.T) {
	dir := t.TempDir()

	ctx := build.Default
	if _, err := ctx.ImportDir(dir, 0); err == nil {
		t.Fatal("ImportDir on empty directory unexpectedly succeeded")
	} else {
		var noGoErr *build.NoGoError
		if !errors.As(err, &noGoErr) {
			t.Fatalf("ImportDir error = %v, want *build.NoGoError", err)
		}
		want := "no buildable Go source files in " + dir
		if got := noGoErr.Error(); got != want {
			t.Fatalf("NoGoError.Error() = %q, want %q", got, want)
		}
	}
}

func TestContextImportAndSrcDirs(t *testing.T) {
	ctx := build.Default
	if build.ToolDir == "" {
		t.Fatal("ToolDir must not be empty")
	}

	dirs := ctx.SrcDirs()
	if len(dirs) == 0 {
		if runtime.GOARCH != "wasm" {
			t.Fatal("SrcDirs returned no directories")
		}
		// The WASI host may not expose the installed GOROOT sources. Check a
		// package inside its preopened temporary directory instead.
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "local.go"), []byte("package localfixture\n"), 0600); err != nil {
			t.Fatal(err)
		}
		pkg, err := ctx.Import(".", dir, build.FindOnly)
		if err != nil {
			t.Fatalf("Context.Import(local) returned error: %v", err)
		}
		if pkg.Dir != dir {
			t.Fatalf("Context.Import(local).Dir = %q, want %q (path=%q, name=%q)", pkg.Dir, dir, pkg.ImportPath, pkg.Name)
		}
		return
	}

	var mode build.ImportMode = build.FindOnly
	if mode&build.FindOnly == 0 {
		t.Fatal("ImportMode FindOnly mask is zero")
	}
	pkg, err := ctx.Import("fmt", "", mode)
	if err != nil {
		t.Fatalf("Context.Import(fmt) returned error: %v", err)
	}
	if pkg.ImportPath != "fmt" {
		t.Fatalf("Import(fmt).ImportPath = %q, want fmt", pkg.ImportPath)
	}

}

func TestPackageDirectivesAndIsCommand(t *testing.T) {
	dir := t.TempDir()

	source := "//go:generate echo hi\npackage main\n\nfunc main() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	ctx := build.Default
	pkg, err := ctx.ImportDir(dir, 0)
	if err != nil {
		t.Fatalf("ImportDir returned error: %v", err)
	}

	if !pkg.IsCommand() {
		t.Fatal("Package.IsCommand() = false, want true")
	}

	if len(pkg.Directives) == 0 {
		t.Fatal("Expected at least one directive in main.go")
	}
	d := pkg.Directives[0]
	if d.Text != "//go:generate echo hi" {
		t.Fatalf("Directive text = %q, want //go:generate echo hi", d.Text)
	}
	if d.Pos.Filename == "" {
		t.Fatal("Directive position filename must be set")
	}
}

func TestArchCharAndIsLocalImport(t *testing.T) {
	if arch, err := build.ArchChar("amd64"); err == nil || arch != "?" {
		t.Fatalf("ArchChar returned (%q, %v), want (\"?\", error)", arch, err)
	} else if !strings.Contains(err.Error(), "no longer used") {
		t.Fatalf("ArchChar error = %v, want message to mention 'no longer used'", err)
	}

	if !build.IsLocalImport("./foo") {
		t.Fatal("IsLocalImport(./foo) = false, want true")
	}
	if build.IsLocalImport("fmt") {
		t.Fatal("IsLocalImport(fmt) = true, want false")
	}
}

func TestMultiplePackageError(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package one\n"), 0o644); err != nil {
		t.Fatalf("write a.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.go"), []byte("package two\n"), 0o644); err != nil {
		t.Fatalf("write b.go: %v", err)
	}

	ctx := build.Default
	if _, err := ctx.ImportDir(dir, 0); err == nil {
		t.Fatal("ImportDir with conflicting packages unexpectedly succeeded")
	} else {
		var mpErr *build.MultiplePackageError
		if !errors.As(err, &mpErr) {
			t.Fatalf("ImportDir error = %v, want *build.MultiplePackageError", err)
		}
		if got := mpErr.Error(); !strings.Contains(got, "found packages") {
			t.Fatalf("MultiplePackageError.Error() = %q, want to mention found packages", got)
		}
	}
}
