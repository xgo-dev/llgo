//go:build !wasm

package gotest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Invalid Go must reach the compiler's diagnostics without a metadata panic.
func TestInvalidMethodReceiverDiagnostics(t *testing.T) {
	compiler := "go"
	if runtime.Compiler == "llgo" {
		compiler = os.Getenv("LLGO_TEST_LLGO")
		if compiler == "" {
			compiler = "llgo"
		}
	}
	for _, tc := range []struct{ file, want string }{
		{"nonlocal.go", "cannot define new methods on non-local type bufio.Reader"},
		{"nonlocal_pointer.go", "cannot define new methods on non-local type bufio.Reader"},
		{"nested_pointer.go", "invalid receiver type"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			source, err := filepath.Abs(filepath.Join("testdata", "invalidreceivers", tc.file))
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			args := []string{"build", "-o", filepath.Join(dir, "invalid.a"), source}
			if runtime.Compiler == "llgo" {
				args = []string{"tool", "compile", "-e", source}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, compiler, args...)
			// Older test toolchains must not read the repository's newer go.mod.
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatalf("compiler timed out: %v\n%s", ctx.Err(), out)
			}
			if err == nil || !strings.Contains(string(out), tc.want) || strings.Contains(string(out), "panic:") {
				t.Fatalf("%s: error = %v, want %q without a panic\n%s", compiler, err, tc.want, out)
			}
		})
	}
}
