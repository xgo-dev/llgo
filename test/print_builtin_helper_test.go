//go:build go1.26 && !wasm

package test

// This host-side probe re-executes the current test binary and captures its
// stderr. WebAssembly has no child-process contract; its builtin print output
// is asserted by the wasm runtime acceptance fixture instead.

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func runBuiltinPrintProbe(t *testing.T) string {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "LLGO_PRINT_HELPER=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Fatalf("builtin print probe failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	return strings.ReplaceAll(stderr.String(), "\r\n", "\n")
}
