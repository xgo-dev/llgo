//go:build !llgo
// +build !llgo

package build

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
)

func TestSelectedSFilesSkipsTestAsm(t *testing.T) {
	dir := "/tmp/pkg"
	got := selectedSFiles(dir, []string{
		"abi_test.s",
		"stub.s",
		"helper.S",
		"compare_test.S",
	})
	want := []string{
		filepath.Join(dir, "stub.s"),
		filepath.Join(dir, "helper.S"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selectedSFiles() = %#v, want %#v", got, want)
	}
}

func TestSelectedSFilesHandlesEmptyInput(t *testing.T) {
	if got := selectedSFiles("", []string{"stub.s"}); got != nil {
		t.Fatalf("selectedSFiles(empty dir) = %#v, want nil", got)
	}
	if got := selectedSFiles("/tmp/pkg", nil); got != nil {
		t.Fatalf("selectedSFiles(nil files) = %#v, want nil", got)
	}
}

func TestShouldSkipPlan9AsmSFilesForTarget(t *testing.T) {
	if !shouldSkipPlan9AsmSFilesForTarget(&Config{Target: "cortex-m-qemu", Goarch: "arm"}, "syscall") {
		t.Fatal("embedded arm syscall asm should be skipped")
	}
	if shouldSkipPlan9AsmSFilesForTarget(&Config{Target: "", Goarch: "arm"}, "syscall") {
		t.Fatal("host arm syscall asm should not be skipped")
	}
	if shouldSkipPlan9AsmSFilesForTarget(&Config{Target: "cortex-m-qemu", Goarch: "arm64"}, "syscall") {
		t.Fatal("arm64 syscall asm should not be skipped by arm-only rule")
	}
	if shouldSkipPlan9AsmSFilesForTarget(&Config{Target: "cortex-m-qemu", Goarch: "arm"}, "internal/bytealg") {
		t.Fatal("only syscall asm should be skipped by embedded arm rule")
	}
}

func TestPkgSFilesUsesPackageLoadDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses a shell script as a fake go command")
	}

	loadDir := t.TempDir()
	expectedLoadDir, err := filepath.EvalSymlinks(loadDir)
	if err != nil {
		t.Fatal(err)
	}
	pkgDir := t.TempDir()
	sfile := filepath.Join(pkgDir, "asm_amd64.s")
	if err := os.WriteFile(sfile, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	binDir := t.TempDir()
	goCmd := filepath.Join(binDir, "go")
	script := `#!/bin/sh
if [ "$PWD" != "$EXPECTED_GO_LIST_DIR" ]; then
	echo "go list ran in $PWD; want $EXPECTED_GO_LIST_DIR" >&2
	exit 1
fi
if [ "$PACKAGE_LOAD_ENV" != "used" ]; then
	echo "go list did not inherit the package load environment" >&2
	exit 1
fi
printf '{"Dir":"%s","SFiles":["asm_amd64.s"]}\n' "$PACKAGE_DIR"
`
	if err := os.WriteFile(goCmd, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("EXPECTED_GO_LIST_DIR", expectedLoadDir)
	t.Setenv("PACKAGE_DIR", pkgDir)

	ctx := &context{
		conf: &packages.Config{
			Dir: loadDir,
			Env: append(os.Environ(), "PACKAGE_LOAD_ENV=used"),
		},
		buildConf: &Config{Goos: "linux", Goarch: "amd64"},
	}
	got, err := pkgSFiles(ctx, &packages.Package{
		ID:      "example.com/asm",
		PkgPath: "example.com/asm",
		Dir:     pkgDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != sfile {
		t.Fatalf("pkgSFiles = %v, want [%s]", got, sfile)
	}
}

func TestPkgSFilesSkipsSyntheticTestMain(t *testing.T) {
	pkgDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(pkgDir, "asm.s"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := &context{
		mode:      ModeTest,
		buildConf: &Config{Goos: "linux", Goarch: "amd64"},
	}
	got, err := pkgSFiles(ctx, &packages.Package{
		ID:      "example.com/p.test",
		PkgPath: "example.com/p.test",
		Name:    "main",
		Dir:     pkgDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("pkgSFiles = %v, want nil", got)
	}
	if cached, ok := ctx.sfilesCache["example.com/p.test"]; !ok || cached != nil {
		t.Fatalf("synthetic test main cache entry = %v, %v; want nil, true", cached, ok)
	}
}

func TestPlan9AsmEnabledInitializesSelectedPackages(t *testing.T) {
	t.Setenv(llgoPlan9ASMPkgs, "example.com/first, example.com/second")
	ctx := &context{buildConf: &Config{}}
	if !ctx.plan9asmEnabled("example.com/first") {
		t.Fatal("selected Plan 9 assembly package was disabled")
	}
	if ctx.plan9asmEnabled("example.com/other") {
		t.Fatal("unselected Plan 9 assembly package was enabled")
	}
	if !ctx.plan9asmReady || ctx.plan9asmMode != plan9asmEnvSelected || !ctx.plan9asmPkgs["example.com/second"] {
		t.Fatalf("prepared Plan 9 assembly policy = mode %v, packages %v", ctx.plan9asmMode, ctx.plan9asmPkgs)
	}
}

func TestPkgSFilesRejectsNilFrozenCache(t *testing.T) {
	ctx := &context{sfilesFrozen: true}
	_, err := pkgSFiles(ctx, &packages.Package{
		ID:      "example.com/unprepared",
		PkgPath: "example.com/unprepared",
	})
	if err == nil {
		t.Fatal("nil frozen SFiles cache accepted an unprepared package")
	}
}
