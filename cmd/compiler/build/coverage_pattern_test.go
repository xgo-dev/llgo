// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.
// See LICENSE for details.

package build

import (
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/packages"
)

func TestMatchLoadedCoveragePackageIncludesTestdata(t *testing.T) {
	cwd := filepath.Join(string(filepath.Separator), "workspace")
	testdata := &packages.Package{
		PkgPath: "example.com/project/test/go/testdata/fixture",
		Dir:     filepath.Join(cwd, "test", "go", "testdata", "fixture"),
	}
	if !matchLoadedCoveragePackage("example.com/project/test/...", cwd, testdata) {
		t.Fatal("import-path pattern omitted a loaded testdata dependency")
	}
	if !matchLoadedCoveragePackage("./test/...", cwd, testdata) {
		t.Fatal("relative pattern omitted a loaded testdata dependency")
	}
	if matchLoadedCoveragePackage("./other/...", cwd, testdata) {
		t.Fatal("unrelated relative pattern matched testdata dependency")
	}
	for _, pattern := range []string{"./test/go/testdata/fixture", "all"} {
		if !matchLoadedCoveragePackage(pattern, cwd, testdata) {
			t.Errorf("%q did not match loaded dependency", pattern)
		}
	}
	for _, pattern := range []string{"std", "cmd", "tool", "work", "./test"} {
		if matchLoadedCoveragePackage(pattern, cwd, testdata) {
			t.Errorf("%q unexpectedly matched loaded dependency", pattern)
		}
	}
}

func TestMatchCoverageImportPattern(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"example.com/project/...", "example.com/project", true},
		{"example.com/project/...", "example.com/project/sub", true},
		{"example.com/project/...", "example.com/other", false},
		{"example.com/project/...", "example.com/project/vendor/dependency", false},
		{"example.com/project/vendor/...", "example.com/project/vendor/dependency", true},
		{"...", "example.com/project/testdata/dependency", true},
		{"vendor/...", "vendor/dependency", false},
		{"vendor/...", "vendor", false},
		{"cmd/...", "cmd/vendor", true},
		{"vendor/...", "vendor/dependency/vendor/nested", false},
		{"...", "a\x00b", false},
		{"a\x00b", "a", false},
		{"\xff", "a", false},
	}
	for _, tc := range tests {
		if got := matchCoverageImportPattern(tc.pattern, tc.path); got != tc.want {
			t.Errorf("matchCoverageImportPattern(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}
