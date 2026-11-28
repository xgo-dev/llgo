package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRuntimeFloatingSourceRejectsDependencies(t *testing.T) {
	path := filepath.Join(runtime.GOROOT(), "src/runtime/complex.go")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, old, replacement, want string }{
		{"missing", "func inf2one", "func renamed", "unreviewed helper"},
		{"signature", "n complex128", "n complex64", "unreviewed helper"},
		{"dependency", "var e, f float64", "var e, f float64; unknownRuntimeHook()", "undefined: unknownRuntimeHook"},
		{"state", "package runtime", "package runtime\nvar state int", "unreviewed var"},
		{"import", "package runtime", "package runtime\nimport _ \"os\"", "unreviewed import"},
		{"directive", "func inf2one", "//go:nosplit\nfunc inf2one", "unreviewed directive"},
		{"build-tag", "package runtime", "//go:build amd64\n\npackage runtime", "unreviewed directive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(string(src), tc.old) {
				t.Fatalf("upstream fixture changed: %s", tc.old)
			}
			read := func(name string) ([]byte, error) {
				if name == path {
					return []byte(strings.Replace(string(src), tc.old, tc.replacement, 1)), nil
				}
				return os.ReadFile(name)
			}
			if _, err := runtimeFloatingPointSources(read, runtime.GOROOT()); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestRuntimeFloatingSourceMissingInputs(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
		err                error
	}{
		{"missing-file", "", "reuse GOROOT runtime/complex.go", os.ErrNotExist},
		{"malformed-file", "package runtime\nfunc (", "expected", nil},
		{"removed-helpers", "package runtime\n", "missing helpers", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			read := func(path string) ([]byte, error) {
				if filepath.Base(path) == "complex.go" {
					return []byte(tc.source), tc.err
				}
				return os.ReadFile(path)
			}
			if _, err := runtimeFloatingPointSources(read, runtime.GOROOT()); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %s", err, tc.want)
			}
		})
	}
}
