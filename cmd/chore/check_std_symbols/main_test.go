package main

import (
	"path/filepath"
	"testing"
)

func TestPkgSpecsSetShorthand(t *testing.T) {
	var specs pkgSpecs
	if err := specs.Set("cmp"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(specs))
	}
	if specs[0].pkgPath != "cmp" {
		t.Fatalf("pkgPath mismatch: %q", specs[0].pkgPath)
	}
	if want := filepath.Join("test", "std", "cmp"); specs[0].testDir != want {
		t.Fatalf("testDir mismatch: got %q want %q", specs[0].testDir, want)
	}
}

func TestPkgSpecsSetExplicit(t *testing.T) {
	var specs pkgSpecs
	if err := specs.Set("cmp=test/std/cmp"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(specs))
	}
	if specs[0].pkgPath != "cmp" {
		t.Fatalf("pkgPath mismatch: %q", specs[0].pkgPath)
	}
	if specs[0].testDir != "test/std/cmp" {
		t.Fatalf("testDir mismatch: %q", specs[0].testDir)
	}
}
