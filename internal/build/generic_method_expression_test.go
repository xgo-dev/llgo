package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// Keep this in ordinary PR CI: the full GOROOT runner is separately gated.
func TestGenericMethodExpressionWrapperIntegration(t *testing.T) {
	source, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "test", "genmeth1.go"))
	if os.IsNotExist(err) {
		t.Skip("selected Go toolchain has no generic-method conformance test")
	}
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, source, 0o600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "genmeth")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if _, err := Do([]string{path}, &Config{Mode: ModeBuild, BuildMode: BuildModeExe, OutFile: bin}); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(bin).CombinedOutput(); err != nil {
		t.Fatalf("generic method conformance: %v\n%s", err, out)
	}
}
