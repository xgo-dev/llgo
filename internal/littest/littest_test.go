package littest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSpecLoadsMarkedSource(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "in.go"), []byte(`// LITTEST
// CHECK: ret void
package main

func main() {}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := LoadSpec(dir)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Mode != ModeFileCheck {
		t.Fatalf("spec.Mode = %v, want %v", spec.Mode, ModeFileCheck)
	}
	if spec.Path != filepath.Join(dir, "in.go") {
		t.Fatalf("spec.Path = %q", spec.Path)
	}
}

func TestLoadSpecReportsMissingDirectory(t *testing.T) {
	_, err := LoadSpec(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("LoadSpec succeeded unexpectedly")
	}
}

func TestLoadSpecRejectsMultipleMarkedSources(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "a.go"), []byte(`// LITTEST
// CHECK: ret void
package main
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(dir, "b.go"), []byte(`// LITTEST
// CHECK: unreachable
package main
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = LoadSpec(dir)
	if err == nil {
		t.Fatal("LoadSpec succeeded unexpectedly")
	}
}

func TestLoadSpecWorksWithSourceChecks(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "in.go"), []byte(`// LITTEST
package main

// CHECK: ret void
func main() {}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := LoadSpec(dir)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Path != filepath.Join(dir, "in.go") {
		t.Fatalf("spec.Path = %q", spec.Path)
	}
}

func TestCheckReportsMalformedDirective(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.go")
	err := os.WriteFile(path, []byte(`// LITTEST
// CHECK: {{[invalid
package main
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
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

func TestFunctionChecksIgnoreFunctionOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.go")
	err := os.WriteFile(path, []byte(`// LITTEST
package main

// DARWIN-ARM64: [[G:@g]] = global i64 1
// DARWIN-ARM64-LABEL: define void @a() {
// DARWIN-ARM64-NEXT: entry:
// DARWIN-ARM64-NEXT:   store i64 1, ptr [[G]]
// DARWIN-ARM64-NEXT: }
func a() {}

// DARWIN-ARM64-LABEL: define void @b() {
// DARWIN-ARM64-NEXT: entry:
// DARWIN-ARM64-NEXT:   store i64 2, ptr [[G]]
// DARWIN-ARM64-NEXT: }
func b() {}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	spec := Spec{Path: path, Mode: ModeFileCheck}
	actual := `@g = global i64 1

define void @b() {
entry:
  store i64 2, ptr @g
}

define void @a() {
entry:
  store i64 1, ptr @g
}
`
	if err := Check(spec, actual, "CHECK", "DARWIN", "ARM64", "DARWIN-ARM64"); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionCheckDoesNotCrossClosingBrace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.go")
	err := os.WriteFile(path, []byte(`// LITTEST
package main

// CHECK-LABEL: define void @first()
// CHECK: call void @only.in.second()
func first() {}

// CHECK-LABEL: define void @second()
// CHECK: call void @only.in.second()
func second() {}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	spec := Spec{Path: path, Mode: ModeFileCheck}
	actual := `define void @first() {
  ret void
}

define void @second() {
  call void @only.in.second()
  ret void
}
`
	if err := Check(spec, actual, "CHECK"); err == nil {
		t.Fatal("a function check crossed the current function's closing brace")
	}
}

func TestFunctionChecksPreserveCrossFunctionBindings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.go")
	err := os.WriteFile(path, []byte(`// LITTEST
package main

// CHECK-LABEL: define ptr @producer() {
// CHECK: ret ptr [[TYPE:@type[0-9]+]]
func producer() {}

// CHECK-LABEL: define i1 @consumer(ptr %0) {
// CHECK: [[MATCH:%[0-9]+]] = icmp eq ptr %0, [[TYPE]]
func consumer() {}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	sections, split, err := functionCheckSections(path, []string{"CHECK"})
	if err != nil {
		t.Fatal(err)
	}
	if split || sections != nil {
		t.Fatalf("cross-function binding was split: %#v", sections)
	}
}

func TestLoadSpecRequiresMarkedSource(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "in.go"), []byte("package main\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = LoadSpec(dir)
	if err == nil {
		t.Fatal("LoadSpec succeeded unexpectedly")
	}
	if !strings.Contains(err.Error(), "missing // LITTEST source lit spec") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHasMarker(t *testing.T) {
	dir := t.TempDir()

	ok, err := HasMarker(filepath.Join(dir, "missing.go"))
	if err == nil || ok {
		t.Fatalf("HasMarker(missing) = (%v, %v)", ok, err)
	}

	empty := filepath.Join(dir, "empty.go")
	err = os.WriteFile(empty, nil, 0644)
	if err != nil {
		t.Fatal(err)
	}
	ok, err = HasMarker(empty)
	if err != nil || ok {
		t.Fatalf("HasMarker(empty) = (%v, %v)", ok, err)
	}

	plain := filepath.Join(dir, "plain.go")
	err = os.WriteFile(plain, []byte("package main\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	ok, err = HasMarker(plain)
	if err != nil || ok {
		t.Fatalf("HasMarker(plain) = (%v, %v)", ok, err)
	}
}

func TestCheck(t *testing.T) {
	checkPath := filepath.Join(t.TempDir(), "check.go")
	if err := os.WriteFile(checkPath, []byte("// CHECK: ok\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		spec Spec
		text string
		want string
	}{
		{
			name: "filecheck match",
			spec: Spec{Path: checkPath, Mode: ModeFileCheck},
			text: "ok\n",
		},
		{
			name: "invalid mode",
			spec: Spec{Path: "bad", Mode: Mode(99)},
			want: "unknown lit spec mode",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Check(tc.spec, tc.text)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("Check succeeded unexpectedly")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoadSpecRequiresMarkerOnFirstLine(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "in.go"), []byte(`
// LITTEST
// CHECK: ret void
package main
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = LoadSpec(dir)
	if err == nil {
		t.Fatal("LoadSpec succeeded unexpectedly")
	}
}

func TestLoadSpecRequiresSlashSlashMarker(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "in.go"), []byte(`# LITTEST
// CHECK: ret void
package main
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = LoadSpec(dir)
	if err == nil {
		t.Fatal("LoadSpec succeeded unexpectedly")
	}
}

func TestLoadSpecIgnoresNonGoFiles(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "in.c"), []byte(`// LITTEST
// CHECK: ret void
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = LoadSpec(dir)
	if err == nil {
		t.Fatal("LoadSpec succeeded unexpectedly")
	}
}
