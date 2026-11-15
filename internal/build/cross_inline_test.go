package build

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

func TestCrossPackageInlineWithCachedDependency(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", repoRoot)
	t.Setenv(llgoBuildCache, "1")
	oldCacheRoot := cacheRootFunc
	cacheDir := t.TempDir()
	cacheRootFunc = func() string { return cacheDir }
	t.Cleanup(func() { cacheRootFunc = oldCacheRoot })
	dir := t.TempDir()
	copyTree(t, filepath.Join("testdata", "crossinline"), dir)
	wantImports := []string{
		"example.com/crossinline/dep.Sum", "example.com/crossinline/dep.Number.Add",
		"reflect.Value.IsValid", "reflect.Value.CanAddr", "reflect.Value.CanSet",
	}
	for phase := range 3 {
		if phase == 2 {
			// Change only the dependency's private implementation. The bridge
			// source is unchanged, but its inlined body must be rebuilt too.
			const old = "const offset = 2"
			for _, name := range []string{"dep/arith.go", "main.go"} {
				path := filepath.Join(dir, name)
				source, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				text := string(source)
				if count := strings.Count(text, old); count != 1 {
					t.Fatalf("%s: found %d occurrences of %q, want 1", name, count, old)
				}
				writeFile(t, path, strings.Replace(text, old, "const offset = 1002", 1))
			}
		}
		conf := NewDefaultConf(ModeBuild)
		conf.OptLevel, conf.LTO = optlevel.O2, lto.Off
		conf.BuildParallelism = 2
		conf.OutFile = filepath.Join(dir, fmt.Sprintf("program-%d", phase))
		if runtime.GOOS == "windows" {
			conf.OutFile += ".exe"
		}
		missingImports := wantImports
		conf.ModuleHook = func(pkg Package) {
			if pkg.PkgPath != "example.com/crossinline" || pkg.LPkg == nil {
				return
			}
			missingImports = nil
			for _, name := range wantImports {
				fn := pkg.LPkg.Module().NamedFunction(name)
				if fn.IsNil() || fn.IsDeclaration() || fn.Linkage() != llvm.AvailableExternallyLinkage {
					missingImports = append(missingImports, name)
				}
			}
		}
		pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
		if err != nil {
			t.Fatal(err)
		}
		if len(missingImports) != 0 {
			t.Fatalf("phase %d: caller lacks dependency inline bodies: %v", phase, missingImports)
		}
		for _, path := range []string{"example.com/crossinline/dep", "example.com/crossinline/bridge", "reflect"} {
			pkg := findPkg(pkgs, path)
			if pkg == nil {
				t.Fatalf("phase %d: missing package %s", phase, path)
			}
			wantCacheHit := phase == 1 || path == "reflect" && phase == 2
			if pkg.CacheHit != wantCacheHit {
				t.Fatalf("phase %d: %s CacheHit=%v, want %v", phase, path, pkg.CacheHit, wantCacheHit)
			}
		}
		if out := strings.TrimSpace(runBinary(t, conf.OutFile)); out != "cross inline ok" {
			t.Fatalf("phase %d: unexpected output: %s", phase, out)
		}
	}
}
