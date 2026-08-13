package filecheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMatchUsesLLVMFileCheck(t *testing.T) {
	spec := `// LITTEST
package main

// CHECK-LABEL: func main
func main() {
	// CHECK: value {{[0-9]+}}
	// CHECK-NEXT: done
}
`
	input := "func main\nvalue 42\ndone\n"
	if err := Match(writeCheckFile(t, spec), input); err != nil {
		t.Fatal(err)
	}
}

func TestMatchSupportsWholeLineAnchors(t *testing.T) {
	spec := `// LITTEST
package main

// CHECK: {{^}}begin{{$}}
// CHECK: {{^}}end{{$}}
`
	input := "begin\nmiddle\nend\n"
	if err := Match(writeCheckFile(t, spec), input); err != nil {
		t.Fatal(err)
	}

	err := Match(writeCheckFile(t, "// CHECK: {{^}}begin{{$}}\n"), "prefix begin\n")
	requireErrContains(t, err, "expected string not found")
}

func TestMatchSupportsSameNotAndEmpty(t *testing.T) {
	spec := `// LITTEST
package main

func main() {
	// CHECK: value
	// CHECK-SAME: 42
	// CHECK-EMPTY:
	// CHECK-NOT: forbidden
	// CHECK: done
}
`
	input := "value 42\n\ndone\n"
	if err := Match(writeCheckFile(t, spec), input); err != nil {
		t.Fatal(err)
	}
}

func TestMatchRejectsForbiddenPattern(t *testing.T) {
	spec := `// LITTEST
package main

func main() {
	// CHECK: ok
	// CHECK-NOT: panic
	// CHECK: done
}
`
	err := Match(writeCheckFile(t, spec), "ok\npanic\ndone\n")
	requireErrContains(t, err, "excluded string found")
}

func TestMatchReportsLLVMErrors(t *testing.T) {
	cases := []struct {
		name  string
		spec  string
		input string
		want  string
	}{
		{
			name: "empty pattern",
			spec: "// CHECK:\n",
			want: "found empty check string",
		},
		{
			name: "unterminated regex",
			spec: "// CHECK: {{[0-9]+\n",
			want: "found start of regex string with no end",
		},
		{
			name: "invalid regex",
			spec: "// CHECK: {{(}}\n",
			want: "invalid regex",
		},
		{
			name: "unknown directive",
			spec: "// CHECK-BOGUS: value\n",
			want: "no check strings found",
		},
		{
			name: "empty pattern not allowed",
			spec: "// CHECK-EMPTY: value\n",
			want: "found non-empty check string for empty check",
		},
		{
			name:  "next without previous check",
			spec:  "// CHECK-NEXT: value\n",
			input: "value\n",
			want:  "without previous 'CHECK: line",
		},
		{
			name:  "next wrong line",
			spec:  "// CHECK: value\n// CHECK-NEXT: done\n",
			input: "value\nother\n",
			want:  "CHECK-NEXT: expected string not found",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Match(writeCheckFile(t, tc.spec), tc.input)
			requireErrContains(t, err, tc.want)
		})
	}
}

func TestMatchSupportsCRLF(t *testing.T) {
	spec := "// LITTEST\r\npackage main\r\n\r\n// CHECK-LABEL: begin\r\n// CHECK-NEXT: done\r\n"
	input := "begin\r\ndone\r\n"
	if err := Match(writeCheckFile(t, spec), input); err != nil {
		t.Fatal(err)
	}
}

func TestMatchWithTargetPrefixes(t *testing.T) {
	spec := `// CHECK: common
// DARWIN: darwin
// ARM64: arm64
// DARWIN-ARM64: combined
// LINUX: linux
`
	input := "common\ndarwin\narm64\ncombined\n"
	if err := MatchWithPrefixes(writeCheckFile(t, spec), input, TargetPrefixes("darwin", "arm64", "")...); err != nil {
		t.Fatal(err)
	}
}

func TestMatchWithTargetPrefixesAllowsUnusedAxes(t *testing.T) {
	prefixes := TargetPrefixes("darwin", "arm64", "")
	if err := MatchWithPrefixes(writeCheckFile(t, "// CHECK: common\n"), "common\n", prefixes...); err != nil {
		t.Fatal(err)
	}
	if err := MatchWithPrefixes(writeCheckFile(t, "// DARWIN-ARM64: specific\n"), "specific\n", prefixes...); err != nil {
		t.Fatal(err)
	}
}

func TestTargetPrefixes(t *testing.T) {
	got := strings.Join(TargetPrefixes("wasip1", "wasm", "wasm"), ",")
	if want := "CHECK,WASIP1,WASM,WASIP1-WASM,TARGET-WASM"; got != want {
		t.Fatalf("TargetPrefixes = %q, want %q", got, want)
	}
	if got := TargetSnapshotPrefix("darwin", "arm64", ""); got != "DARWIN-ARM64" {
		t.Fatalf("TargetSnapshotPrefix = %q, want DARWIN-ARM64", got)
	}
	if got := TargetSnapshotPrefix("linux", "386", ""); got != "LINUX-ARCH-386" {
		t.Fatalf("TargetSnapshotPrefix = %q, want LINUX-ARCH-386", got)
	}
}

func writeCheckFile(t *testing.T, spec string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.go")
	if err := os.WriteFile(path, []byte(spec), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func requireErrContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err, want)
	}
}
