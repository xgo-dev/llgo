package littest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadSpecSelectsMarkedSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.go")
	writeFile(t, path, `// LITTEST
// CHECK: ret void
package main
`)

	spec, err := LoadSpec(dir)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Path != path {
		t.Fatalf("spec.Path = %q, want %q", spec.Path, path)
	}
	if spec.PostABI {
		t.Fatal("plain LITTEST marker unexpectedly selected post-ABI IR")
	}
}

func TestLoadSpecSelectsPostABIIR(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "in.go"), `// LITTEST: POST-ABI
package main

// CHECK: sret
func main() {}
`)

	spec, err := LoadSpec(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !spec.PostABI {
		t.Fatal("POST-ABI marker did not select post-ABI IR")
	}
}

func TestLoadSpecSelectsPostABITargets(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "in.go"), `// LITTEST: POST-ABI linux/amd64 linux/arm64
package main

// CHECK: ret void
func main() {}
`)

	spec, err := LoadSpec(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !spec.PostABI {
		t.Fatal("target matrix did not select post-ABI IR")
	}
	got := make([]string, len(spec.Targets))
	for i, target := range spec.Targets {
		got[i] = target.String()
	}
	if strings.Join(got, ",") != "linux/amd64,linux/arm64" {
		t.Fatalf("targets = %v, want [linux/amd64 linux/arm64]", got)
	}
}

func TestLoadSpecSelectsDefaultStageTargets(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "in.go"), `// LITTEST darwin/arm64 linux/amd64
package main

// CHECK: ret void
func main() {}
`)

	spec, err := LoadSpec(dir)
	if err != nil {
		t.Fatal(err)
	}
	if spec.PostABI {
		t.Fatal("default marker unexpectedly selected post-ABI IR")
	}
	got := make([]string, len(spec.Targets))
	for i, target := range spec.Targets {
		got[i] = target.String()
	}
	if strings.Join(got, ",") != "darwin/arm64,linux/amd64" {
		t.Fatalf("targets = %v, want [darwin/arm64 linux/amd64]", got)
	}
}

func TestLoadSpecRejectsInvalidTargets(t *testing.T) {
	tests := []struct {
		marker string
		want   string
	}{
		{marker: "// LITTEST: POST-ABI amd64", want: `invalid LITTEST target "amd64"`},
		{marker: "// LITTEST: POST-ABI linux/amd64 linux/amd64", want: `duplicate LITTEST target "linux/amd64"`},
		{marker: "// LITTEST amd64", want: `invalid LITTEST target "amd64"`},
	}
	for _, test := range tests {
		t.Run(test.want, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "in.go"), test.marker+"\npackage main\n")

			_, err := LoadSpec(dir)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("LoadSpec error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestLoadSpecReportsMissingDirectory(t *testing.T) {
	_, err := LoadSpec(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("LoadSpec succeeded unexpectedly")
	}
}

func TestLoadSpecRequiresMarkedSource(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "in.go"), "package main\n")
	writeFile(t, filepath.Join(dir, "out.ll"), "legacy literal IR")

	_, err := LoadSpec(dir)
	if err == nil {
		t.Fatal("LoadSpec accepted legacy out.ll unexpectedly")
	}
	if !strings.Contains(err.Error(), "missing // LITTEST source lit spec") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSpecRejectsMultipleMarkedSources(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.go"), `// LITTEST
// CHECK: ret void
package main
`)
	writeFile(t, filepath.Join(dir, "b.go"), `// LITTEST
// CHECK: unreachable
package main
`)

	_, err := LoadSpec(dir)
	if err == nil {
		t.Fatal("LoadSpec succeeded unexpectedly")
	}
}

func TestCheck(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.go")
	writeFile(t, path, "// LITTEST\n// CHECK: ret void\npackage main\n")
	spec, err := LoadSpec(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := Check(spec, "ret void\n"); err != nil {
		t.Fatal(err)
	}
	if err := Check(spec, "ret void\n", "CHECK", "DARWIN-ARM64"); err != nil {
		t.Fatal(err)
	}
	if err := Check(spec, "unreachable\n"); err == nil {
		t.Fatal("Check accepted mismatching IR unexpectedly")
	}
}

func TestCheckReportsMalformedDirective(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "in.go"), `// LITTEST
// CHECK: {{[invalid
package main
`)
	spec, err := LoadSpec(dir)
	if err != nil {
		t.Fatal(err)
	}
	err = Check(spec, "")
	if err == nil {
		t.Fatal("Check succeeded unexpectedly")
	}
	if !strings.Contains(err.Error(), "found start of regex string with no end") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHasMarker(t *testing.T) {
	dir := t.TempDir()

	ok, err := HasMarker(filepath.Join(dir, "missing.go"))
	if err == nil || ok {
		t.Fatalf("HasMarker(missing) = (%v, %v)", ok, err)
	}

	tests := []struct {
		name     string
		contents string
		want     bool
	}{
		{name: "empty"},
		{name: "plain", contents: "package main\n"},
		{name: "marker", contents: "// LITTEST\npackage main\n", want: true},
		{name: "default targets", contents: "// LITTEST darwin/arm64 linux/amd64\npackage main\n", want: true},
		{name: "post abi", contents: "// LITTEST: POST-ABI\npackage main\n", want: true},
		{name: "post abi targets", contents: "// LITTEST: POST-ABI linux/amd64 linux/arm64\npackage main\n", want: true},
		{name: "not first line", contents: "\n// LITTEST\npackage main\n"},
		{name: "wrong comment", contents: "# LITTEST\npackage main\n"},
		{name: "unknown marker", contents: "// LITTEST: UNKNOWN\npackage main\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, strings.ReplaceAll(tc.name, " ", "_")+".go")
			writeFile(t, path, tc.contents)
			got, err := HasMarker(path)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("HasMarker() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFindMarkedSourceFileReportsUnreadableSource(t *testing.T) {
	dir := t.TempDir()
	if err := os.Symlink("missing.go", filepath.Join(dir, "broken.go")); err != nil {
		t.Skipf("Symlink is unavailable: %v", err)
	}

	_, _, err := FindMarkedSourceFile(dir)
	if err == nil {
		t.Fatal("FindMarkedSourceFile succeeded unexpectedly")
	}
}

func TestFindMarkedSourceFileIgnoresNonSourceFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "in.c"), "// LITTEST\n")
	writeFile(t, filepath.Join(dir, "in_test.go"), "// LITTEST\npackage main\n")

	path, ok, err := FindMarkedSourceFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ok || path != "" {
		t.Fatalf("FindMarkedSourceFile() = %q, %v, want no spec", path, ok)
	}
}
