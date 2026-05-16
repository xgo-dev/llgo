package cltest

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/build"
	"github.com/goplus/llgo/internal/metadata"
	"github.com/qiniu/x/test"
)

// TestMetaFromDir runs metadata golden-file tests for all subdirectories
// under relDir. Each subdirectory should contain in.go and meta-expect.txt.
func TestMetaFromDir(t *testing.T, relDir string) {
	rootDir, err := os.Getwd()
	if err != nil {
		t.Fatal("Getwd failed:", err)
	}
	dir := filepath.Join(rootDir, relDir)
	fis, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal("ReadDir failed:", err)
	}
	for _, fi := range fis {
		name := fi.Name()
		if !fi.IsDir() || strings.HasPrefix(name, "_") {
			continue
		}
		t.Run(name, func(t *testing.T) {
			testMetaDir(t, filepath.Join(dir, name))
		})
	}
}

func testMetaDir(t *testing.T, dir string) {
	t.Helper()

	var pm *metadata.PackageMeta
	conf := build.NewDefaultConf(build.ModeRun)
	conf.ModuleHook = func(pkg build.Package) {
		lpkg := pkg.LPkg
		if lpkg == nil {
			return
		}
		// Only capture metadata for the package whose source files
		// are in the test directory.
		if slices.ContainsFunc(pkg.Package.GoFiles, func(file string) bool {
			return filepath.Dir(file) == dir
		}) {
			pm = lpkg.Meta()
		}
	}

	_, err := runWithConf(".", dir, conf)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	if pm == nil {
		t.Fatal("no PackageMeta captured for test package")
	}

	got := metadata.MetaString(pm)
	expectFile := filepath.Join(dir, "meta-expect.txt")
	expect, err := os.ReadFile(expectFile)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.WriteFile(expectFile, []byte(got), 0644); err != nil {
				t.Fatalf("WriteFile %s failed: %v", expectFile, err)
			}
			t.Logf("wrote initial golden file: %s", expectFile)
			return
		}
		t.Fatalf("ReadFile %s failed: %v", expectFile, err)
	}

	expected := string(expect)
	if got != expected {
		newFile := filepath.Join(dir, "meta-expect.txt.new")
		_ = os.WriteFile(newFile, []byte(got), 0644)
		fmt.Printf("Meta diff:\ngot:\n%s\nvs expected:\n%s\n", got, expected)
		_ = test.Diff(t, filepath.Join(dir, "meta-expect.txt"), []byte(got), expect)
		t.Fatal("unexpected metadata output")
	}
}

// CaptureMeta compiles the Go package at pkgDir and returns the formatted
// PackageMeta string. It is used by gentests to regenerate golden files.
func CaptureMeta(pkgDir string) (string, error) {
	var pm *metadata.PackageMeta
	conf := build.NewDefaultConf(build.ModeRun)
	conf.ModuleHook = func(pkg build.Package) {
		lpkg := pkg.LPkg
		if lpkg == nil {
			return
		}
		if slices.ContainsFunc(pkg.Package.GoFiles, func(file string) bool {
			return filepath.Dir(file) == pkgDir
		}) {
			pm = lpkg.Meta()
		}
	}

	_, err := runWithConf(".", pkgDir, conf)
	if err != nil {
		return "", fmt.Errorf("build failed: %w", err)
	}

	if pm == nil {
		return "", fmt.Errorf("no PackageMeta captured for package at %s", pkgDir)
	}

	return metadata.MetaString(pm), nil
}
