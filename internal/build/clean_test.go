//go:build !llgo

package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
)

func TestCleanMainPkgRemovesPCLNSidecars(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	sourceDir := filepath.Join(root, "source")
	for _, dir := range []string{binDir, sourceDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	pkg := &packages.Package{
		PkgPath:         "example.com/tools/demo",
		CompiledGoFiles: []string{filepath.Join(sourceDir, "main.go")},
	}
	conf := &Config{BinPath: binDir, AppExt: ".exe"}
	outputs := []string{
		filepath.Join(binDir, "demo.exe"),
		filepath.Join(sourceDir, "demo.exe"),
	}
	for _, executable := range outputs {
		for _, artifact := range []string{executable, pclnSidecarPath(executable)} {
			if err := os.WriteFile(artifact, []byte("artifact"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	keep := filepath.Join(sourceDir, "demo.exe.pclntab.keep")
	if err := os.WriteFile(keep, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	cleanMainPkg(pkg, conf, false)

	for _, executable := range outputs {
		for _, artifact := range []string{executable, pclnSidecarPath(executable)} {
			if _, err := os.Stat(artifact); !os.IsNotExist(err) {
				t.Errorf("artifact %q still exists (stat error %v)", artifact, err)
			}
		}
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("unrelated file was removed: %v", err)
	}
}
