//go:build !llgo
// +build !llgo

package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goplus/llgo/internal/cabi"
	"github.com/goplus/llgo/internal/packages"
)

func TestDirHasAsmFile(t *testing.T) {
	dir := t.TempDir()
	if dirHasAsmFile(dir) {
		t.Fatal("empty directory should not have asm files")
	}
	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if dirHasAsmFile(dir) {
		t.Fatal("directory with no .s/.S files should not have asm files")
	}
	if err := os.WriteFile(filepath.Join(dir, "lower.s"), []byte("TEXT ·x(SB),$0-0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !dirHasAsmFile(dir) {
		t.Fatal("directory with .s file should have asm files")
	}

	upperDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(upperDir, "upper.S"), []byte("TEXT ·x(SB),$0-0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !dirHasAsmFile(upperDir) {
		t.Fatal("directory with .S file should have asm files")
	}
	if !dirHasAsmFile(filepath.Join(t.TempDir(), "missing")) {
		t.Fatal("unreadable/missing directory should fall back to source selection")
	}
}

func TestPkgSFilesUsesOtherFiles(t *testing.T) {
	dir := t.TempDir()
	pkg := &packages.Package{
		ID:         "example.com/otherfiles",
		PkgPath:    "example.com/otherfiles",
		Dir:        dir,
		OtherFiles: []string{filepath.Join(dir, "selected.s"), filepath.Join(dir, "note.txt")},
	}
	ctx := &context{conf: &packages.Config{}, buildConf: &Config{Goos: "linux", Goarch: "amd64"}}
	files, err := pkgSFiles(ctx, pkg)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "selected.s")
	if len(files) != 1 || files[0] != want {
		t.Fatalf("pkgSFiles returned %v, want [%s]", files, want)
	}
	if got := ctx.sfilesCache[pkg.ID]; len(got) != 1 || got[0] != want {
		t.Fatalf("sfiles cache = %v, want [%s]", got, want)
	}
}

func TestPkgSFilesChacha8UsesStubBeforeOtherFiles(t *testing.T) {
	dir := t.TempDir()
	stub := filepath.Join(dir, "chacha8_stub.s")
	if err := os.WriteFile(stub, []byte("TEXT ·stub(SB),$0-0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	optimized := filepath.Join(dir, "chacha8_amd64.s")
	pkg := &packages.Package{
		ID:         "internal/chacha8rand",
		PkgPath:    "internal/chacha8rand",
		Dir:        dir,
		OtherFiles: []string{optimized},
	}
	ctx := &context{conf: &packages.Config{}, buildConf: &Config{Goos: "linux", Goarch: "amd64"}}
	files, err := pkgSFiles(ctx, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != stub {
		t.Fatalf("pkgSFiles returned %v, want stub %s", files, stub)
	}
}

func TestPkgSFilesUsesBuildContextSelection(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "selected_amd64.s"), []byte("TEXT ·selected(SB),$0-0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skipped_arm64.s"), []byte("TEXT ·skipped(SB),$0-0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := &context{conf: &packages.Config{}, buildConf: &Config{Goos: "linux", Goarch: "amd64"}}
	pkg := &packages.Package{ID: "example.com/asmselect", PkgPath: "example.com/asmselect", Dir: dir}
	files, err := pkgSFiles(ctx, pkg)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "selected_amd64.s")
	if len(files) != 1 || files[0] != want {
		t.Fatalf("pkgSFiles returned %v, want [%s]", files, want)
	}
	if has, ok := ctx.asmDirCache[dir]; !ok || !has {
		t.Fatalf("asm dir cache[%s] = %v, %v; want true", dir, has, ok)
	}
}

func TestPkgSFilesCachesNoAsmResult(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := &context{conf: &packages.Config{}, buildConf: &Config{Goos: "linux", Goarch: "amd64"}}
	pkg := &packages.Package{ID: "example.com/noasm", PkgPath: "example.com/noasm", Dir: dir}
	files, err := pkgSFiles(ctx, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("pkgSFiles returned %v, want no files", files)
	}
	if ctx.sfilesCache == nil {
		t.Fatal("sfiles cache was not initialized")
	}
	if _, ok := ctx.sfilesCache[pkg.ID]; !ok {
		t.Fatal("no-asm result was not cached")
	}
	if has, ok := ctx.asmDirCache[dir]; !ok || has {
		t.Fatalf("asm dir cache[%s] = %v, %v; want false", dir, has, ok)
	}
}

func TestHasAltPkgForTarget_AllowsAdditivePatchWithPlan9Asm(t *testing.T) {
	conf := &Config{Goarch: "arm64", AbiMode: cabi.ModeAllFunc}
	if !plan9asmEnabledByDefault(conf, "internal/runtime/sys") {
		t.Fatal("plan9asm should be enabled by default for internal/runtime/sys on arm64")
	}
	if !hasAltPkgForTarget(conf, "internal/runtime/sys") {
		t.Fatal("internal/runtime/sys should keep its additive alt package even when plan9asm is enabled")
	}
}
