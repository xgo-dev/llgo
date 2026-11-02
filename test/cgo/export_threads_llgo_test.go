//go:build llgo && !wasm

package cgo

import (
	"os"
	"os/exec"
	"testing"
)

func buildCExportFixture(t *testing.T, dir, output string, mode ...string) {
	t.Helper()
	compiler := os.Getenv("LLGO_TEST_LLGO")
	if compiler == "" {
		compiler = "llgo"
	}
	args := []string{"build", "-o", output}
	if len(mode) != 0 {
		args = append(args, "-buildmode="+mode[0])
	}
	cmd := exec.Command(compiler, append(args, ".")...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build C-export fixture: %v\n%s", err, out)
	}
}
