//go:build !llgo
// +build !llgo

package build

import (
	"path/filepath"
	"reflect"
	"testing"
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
