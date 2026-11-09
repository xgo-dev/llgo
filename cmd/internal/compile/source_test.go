package compile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompilerSourceChecks(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
		opts               options
	}{
		{"complete", "func F()", "missing function body", options{complete: true}},
		{"external", "func F()", "", options{}},
		{"complete-linkname", "import _ \"unsafe\"\n//go:linkname F C.f\nfunc F()", "", options{complete: true}},
		{"complete-wasmimport", "//go:wasmimport env f\nfunc F()", "", options{complete: true}},
		{"keepalive", "//go:uintptrkeepalive\nfunc F(uintptr) {}", "go:uintptrkeepalive requires go:nosplit", options{standard: true}},
		{"keepalive-nosplit", "//go:uintptrkeepalive\n//go:nosplit\nfunc F(uintptr) {}", "", options{standard: true}},
		{"keepalive-escape", "//go:uintptrkeepalive\n//go:uintptrescapes\nfunc F(uintptr) {}", "", options{standard: true}},
		{"keepalive-external", "//go:uintptrkeepalive\nfunc F(uintptr)", "", options{standard: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "input.go")
			if err := os.WriteFile(path, []byte("package p\n"+tc.source+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			err := validateCompilerSources([]string{path}, &tc.opts)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "input.go:") || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want positioned %q, got %v", tc.want, err)
			}
		})
	}
}
