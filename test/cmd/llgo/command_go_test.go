//go:build !llgo && !wasm

package llgocmd

import (
	"os"
	"os/exec"
	"testing"
)

const toolCompilerName = "go"

func runToolCompile(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("go", append([]string{"tool", "compile"}, args...)...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func runCompiler(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	return runGoCompiler(t, dir, args...)
}
