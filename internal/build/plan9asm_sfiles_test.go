//go:build !llgo
// +build !llgo

package build

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/goplus/llgo/internal/packages"
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
