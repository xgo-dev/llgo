//go:build llgo && !wasm

package llgocmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

var (
	llgoOnce sync.Once
	llgoDir  string
	llgoPath string
	llgoErr  string
)

const toolCompilerName = "llgo"

func runToolCompile(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(testLLGo(t), append([]string{"tool", "compile"}, args...)...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func runCompiler(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	return runCompilerCommand(t, dir, testLLGo(t), args...)
}

func TestMain(m *testing.M) {
	code := m.Run()
	if llgoDir != "" {
		if err := os.RemoveAll(llgoDir); err != nil {
			fmt.Fprintf(os.Stderr, "remove temporary llgo command: %v\n", err)
			code = 1
		}
	}
	os.Exit(code)
}

func testLLGo(t *testing.T) string {
	t.Helper()
	root := repositoryRoot(t)
	t.Setenv("LLGO_ROOT", root)
	// These overrides are trusted inputs supplied by the developer or CI
	// environment running the test suite.
	for _, name := range []string{"LLGO_TEST_COMPILER", "LLGO_TEST_LLGO", "LLGO"} {
		if path := os.Getenv(name); path != "" {
			absolute, err := filepath.Abs(path)
			if err != nil {
				t.Fatalf("resolve %s: %v", name, err)
			}
			return absolute
		}
	}

	llgoOnce.Do(func() {
		dir, err := os.MkdirTemp("", "llgo-command-test-")
		if err != nil {
			llgoErr = err.Error()
			return
		}
		llgoDir = dir
		llgoPath = executablePath(dir, "llgo")
		cmd := exec.Command("go", "build", "-o", llgoPath, "./cmd/llgo")
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			llgoErr = err.Error() + "\n" + string(output)
		}
	})
	if llgoErr != "" {
		t.Fatalf("build llgo: %s", llgoErr)
	}
	return llgoPath
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root not found")
		}
		dir = parent
	}
}
