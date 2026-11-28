package build

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/lto"
	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	"github.com/xgo-dev/llvm"
)

func TestCrossPackageInlineWithCachedDependency(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
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
		"runtime.(*Func).Name",
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

func TestCrossPackageInlineDeclarationOnly(t *testing.T) {
	t.Setenv(llgoBuildCache, "0")
	for _, level := range []optlevel.Level{optlevel.O0, optlevel.O2, optlevel.O3} {
		t.Run(level.String(), func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.OptLevel, conf.LTO = level, lto.Off
			conf.OutFile = filepath.Join(t.TempDir(), "program")
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			if _, err := Do([]string{"./testdata/crossinlinedecl"}, conf); err != nil {
				t.Fatal(err)
			}
			if got := strings.TrimSpace(runBinary(t, conf.OutFile)); got != "42" {
				t.Fatalf("linked method returned %s, want 42", got)
			}
		})
	}
}

func TestCrossPackageInlineLTOPolicy(t *testing.T) {
	t.Setenv(llgoBuildCache, "0")
	for _, mode := range []lto.Mode{lto.Off, lto.Thin, lto.Full} {
		t.Run(mode.String(), func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.OptLevel, conf.LTO = optlevel.Os, mode
			conf.BuildParallelism = 2
			conf.OutFile = filepath.Join(t.TempDir(), "program")
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			checked := false
			conf.ModuleHook = func(pkg Package) {
				if pkg.PkgPath != "example.com/crossinline" || pkg.LPkg == nil {
					return
				}
				checked = true
				for _, name := range []string{"example.com/crossinline/bridge.Sum", "example.com/crossinline/dep.Number.Add", "runtime.(*Func).Name"} {
					fn := pkg.LPkg.Module().NamedFunction(name)
					if fn.IsNil() || fn.IsDeclaration() != (mode == lto.Full) {
						t.Errorf("%s: unexpected body import with %s LTO", name, mode)
					}
				}
			}
			if _, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: filepath.Join("testdata", "crossinline")}); err != nil {
				t.Fatal(err)
			}
			if !checked {
				t.Fatal("caller module not checked")
			}
			if got := strings.TrimSpace(runBinary(t, conf.OutFile)); got != "cross inline ok" {
				t.Fatalf("unexpected output: %s", got)
			}
		})
	}
}
