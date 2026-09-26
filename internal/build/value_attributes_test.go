//go:build !llgo

package build

import (
	"github.com/xgo-dev/llvm"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValueAttributesCache(t *testing.T) {
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
	fixture := filepath.Join(root, "internal", "build", "testdata", "valueattrs")
	depFile := filepath.Join(fixture, "dep", "dep.go")
	original, err := os.ReadFile(depFile)
	if err != nil {
		t.Fatal(err)
	}
	const prefix = "github.com/xgo-dev/llgo/internal/build/testdata/valueattrs"
	for _, phase := range []struct {
		name           string
		annotated, hit bool
	}{{"cold", true, false}, {"hit", true, true}, {"removed", false, false}} {
		t.Run(phase.name, func(t *testing.T) {
			source := string(original)
			if !phase.annotated {
				source = strings.ReplaceAll(source, "//llgo:result nonnull sameas(p)", "")
			}
			conf := NewDefaultConf(ModeBuild)
			conf.OutFile = filepath.Join(t.TempDir(), "attrs")
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			conf.Overlay = map[string][]byte{depFile: []byte(source)}
			checked := false
			conf.ModuleHook = func(pkg Package) {
				if pkg.LPkg == nil || pkg.PkgPath != prefix {
					return
				}
				fn := pkg.LPkg.Module().NamedFunction(prefix + "/dep.Checked")
				if fn.IsNil() {
					t.Error("missing imported declaration")
					return
				}
				got := !fn.GetEnumAttributeAtIndex(0, llvm.AttributeKindID("nonnull")).IsNil()
				if got != phase.annotated {
					t.Errorf("nonnull = %v, want %v", got, phase.annotated)
				}
				checked = true
			}
			pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: fixture})
			if err != nil {
				t.Fatal(err)
			}
			if !checked {
				t.Fatal("caller module not checked")
			}
			found := false
			for _, pkg := range pkgs {
				if pkg.PkgPath == prefix+"/dep" {
					found = true
					if pkg.CacheHit != phase.hit {
						t.Errorf("CacheHit=%v want %v", pkg.CacheHit, phase.hit)
					}
				}
			}
			if !found {
				t.Fatal("dependency not built")
			}
			if out, err := exec.Command(conf.OutFile).CombinedOutput(); err != nil {
				t.Fatalf("execution: %v\n%s", err, out)
			}
		})
	}
}
