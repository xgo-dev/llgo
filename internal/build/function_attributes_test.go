//go:build !llgo

package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/xgo-dev/llvm"
)

// Rebuild the caller even when the dependency is cached. Its external function
// declarations must reflect source annotations, including annotation-only edits.
func TestFunctionAttributesCacheAndUnwind(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	t.Setenv(llgoBuildCache, "1")
	oldRoot := cacheRootFunc
	cacheDir := t.TempDir()
	cacheRootFunc = func() string { return cacheDir }
	defer func() { cacheRootFunc = oldRoot }()
	fixture := filepath.Join(root, "internal", "build", "testdata", "functionattrs")
	depFile := filepath.Join(fixture, "dep", "dep.go")
	original, err := os.ReadFile(depFile)
	if err != nil {
		t.Fatal(err)
	}
	// Exercise Windows checkout line endings on every host.
	original = []byte(strings.ReplaceAll(strings.ReplaceAll(string(original), "\r\n", "\n"), "\n", "\r\n"))
	const prefix = "github.com/xgo-dev/llgo/internal/build/testdata/functionattrs"
	for _, phase := range []struct {
		name      string
		annotated bool
		hit       int // -1 permits either a restored artifact or recompilation.
		tags      string
	}{
		{"first", true, 0, ""}, {"hit", true, 1, ""},
		{"removed", false, 0, ""}, {"restored", true, -1, ""},
		{"restored-hit", true, 1, ""}, {"nogc", true, 0, "nogc"},
	} {
		t.Run(phase.name, func(t *testing.T) {
			source := string(original)
			if !phase.annotated {
				source = strings.ReplaceAll(source, "//llgo:cold", "")
				source = strings.ReplaceAll(source, "//llgo:noreturn", "")
			}
			conf := NewDefaultConf(ModeBuild)
			conf.Tags = phase.tags
			conf.OutFile = filepath.Join(t.TempDir(), "attrs")
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			conf.BuildParallelism = 2
			conf.Overlay = map[string][]byte{depFile: []byte(source)}
			var mu sync.Mutex
			checked := false
			conf.ModuleHook = func(pkg Package) {
				if pkg.PkgPath != prefix || pkg.LPkg == nil {
					return
				}
				mu.Lock()
				defer mu.Unlock()
				for _, symbol := range []string{"Stop", "Generic[int]", "T.Stop"} {
					fn := pkg.LPkg.Module().NamedFunction(prefix + "/dep." + symbol)
					if fn.IsNil() {
						t.Errorf("missing imported %s declaration", symbol)
						continue
					}
					for _, name := range []string{"cold", "noreturn"} {
						got := !fn.GetEnumFunctionAttribute(llvm.AttributeKindID(name)).IsNil()
						if got != phase.annotated {
							t.Errorf("%s: imported %s = %v, want %v", symbol, name, got, phase.annotated)
						}
					}
				}
				checked = true
			}
			pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: fixture})
			if err != nil {
				t.Fatal(err)
			}
			if !checked {
				t.Fatal("caller module was not checked")
			}
			found := false
			for _, pkg := range pkgs {
				if pkg.PkgPath == prefix+"/dep" {
					found = true
					if phase.hit >= 0 && pkg.CacheHit != (phase.hit == 1) {
						t.Errorf("dependency CacheHit = %v, want %v", pkg.CacheHit, phase.hit == 1)
					}
				}
			}
			if !found {
				t.Fatal("missing dependency build result")
			}
			if out, err := exec.Command(conf.OutFile).CombinedOutput(); err != nil {
				t.Fatalf("panic/defer/recover execution: %v\n%s", err, out)
			}
		})
	}
}
