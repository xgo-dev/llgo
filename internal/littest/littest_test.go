package littest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSpecPrefersMarkedSourceOverOutLL(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "in.go"), []byte(`// LITTEST
// CHECK: ret void
package main

func main() {}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(dir, "out.ll"), []byte(`; ModuleID = 'main'
define void @main() {
  ret void
}
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

func TestLoadSpecWorksWithoutOutLLWhenSourceChecksExist(t *testing.T) {
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

func TestLoadSpecRejectsMarkedSourceWithoutDirectives(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "in.go"), []byte(`// LITTEST
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

func TestLoadSpecReportsMalformedDirective(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "in.go"), []byte(`// LITTEST
// CHECK: {{[invalid
package main
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = LoadSpec(dir)
	if err == nil {
		t.Fatal("LoadSpec succeeded unexpectedly")
	}
	if !strings.Contains(err.Error(), "unterminated '{{' in pattern") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSpecFallsBackToOutLLWithoutMarker(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "in.go"), []byte(`// CHECK: ret void
package main
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(dir, "out.ll"), []byte(`; ModuleID = 'main'
define void @main() {
  ret void
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := LoadSpec(dir)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Path != filepath.Join(dir, "out.ll") {
		t.Fatalf("spec.Path = %q", spec.Path)
	}
	if spec.Mode != ModeLiteral {
		t.Fatalf("spec.Mode = %v", spec.Mode)
	}
}

func TestLoadSpecReportsMissingOutLL(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "in.go"), []byte("package main\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = LoadSpec(dir)
	if err == nil {
		t.Fatal("LoadSpec succeeded unexpectedly")
	}
}

func TestLoadSpecSupportsSkipOutLL(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "out.ll"), []byte(`;`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := LoadSpec(dir)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Mode != ModeSkip {
		t.Fatalf("spec.Mode = %v", spec.Mode)
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
	cases := []struct {
		name string
		spec Spec
		text string
		want string
	}{
		{
			name: "skip",
			spec: Spec{Path: "skip", Mode: ModeSkip},
		},
		{
			name: "literal match",
			spec: Spec{Path: "literal", Text: "ok", Mode: ModeLiteral},
			text: "ok",
		},
		{
			name: "literal mismatch",
			spec: Spec{Path: "literal", Text: "ok", Mode: ModeLiteral},
			text: "bad",
			want: "literal LLVM IR mismatch",
		},
		{
			name: "filecheck match",
			spec: Spec{Path: "check.go", Text: "// CHECK: ok\n", Mode: ModeFileCheck},
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
	err = os.WriteFile(filepath.Join(dir, "out.ll"), []byte(`; ModuleID = 'main'
define void @main() {
  ret void
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := LoadSpec(dir)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Path != filepath.Join(dir, "out.ll") {
		t.Fatalf("spec.Path = %q", spec.Path)
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
	err = os.WriteFile(filepath.Join(dir, "out.ll"), []byte(`; ModuleID = 'main'
define void @main() {
  ret void
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := LoadSpec(dir)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Path != filepath.Join(dir, "out.ll") {
		t.Fatalf("spec.Path = %q", spec.Path)
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
	err = os.WriteFile(filepath.Join(dir, "out.ll"), []byte(`; ModuleID = 'main'
define void @main() {
  ret void
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := LoadSpec(dir)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Path != filepath.Join(dir, "out.ll") {
		t.Fatalf("spec.Path = %q", spec.Path)
	}
}
