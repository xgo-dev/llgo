package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// A re-exported constant must follow the parser's source view on both cache
// misses and hits, including a change to only the runtime source patch.
func TestRuntimeCompilerPackageCache(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	t.Setenv(llgoBuildCache, "1")
	oldCacheRoot := cacheRootFunc
	cacheDir := t.TempDir()
	cacheRootFunc = func() string { return cacheDir }
	t.Cleanup(func() { cacheRootFunc = oldCacheRoot })
	const identityPath = "github.com/xgo-dev/llgo/internal/build/testdata/runtimecompiler/identity"
	fingerprints := make(map[string]string)
	for phase, tc := range []struct {
		name   string
		branch string
		hit    bool
	}{
		{"llgo", "llgo-branch", false},
		{"llgo", "llgo-branch", true},
		{"llgo-cache", "other-branch", false},
		{"llgo-cache", "other-branch", true},
		{"llgo", "llgo-branch", true},
	} {
		t.Run(fmt.Sprintf("%d-%s", phase, tc.name), func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.BuildParallelism = 2
			conf.OutFile = filepath.Join(t.TempDir(), "compiler")
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			if tc.name != "llgo" {
				conf.Overlay = runtimeCompilerOverlay(t, root, tc.name)
			}
			pkgs, err := Build(Invocation{
				Args: []string{"."}, Config: conf,
				Dir: filepath.Join(root, "internal", "build", "testdata", "runtimecompiler"),
			})
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, pkg := range pkgs {
				if pkg.PkgPath != identityPath {
					continue
				}
				found = true
				if pkg.CacheHit != tc.hit {
					t.Errorf("re-exporting package CacheHit = %v, want %v", pkg.CacheHit, tc.hit)
				}
				if previous, ok := fingerprints[tc.name]; ok && previous != pkg.Fingerprint {
					t.Error("restoring the runtime source view did not restore its importer's fingerprint")
				}
				if tc.name != "llgo" && pkg.Fingerprint == fingerprints["llgo"] {
					t.Error("runtime constant change did not invalidate its importer")
				}
				fingerprints[tc.name] = pkg.Fingerprint
			}
			if !found {
				t.Fatal("missing re-exporting package")
			}
			output, err := exec.Command(conf.OutFile).CombinedOutput()
			if err != nil {
				t.Fatalf("run: %v\n%s", err, output)
			}
			want := fmt.Sprintf("%s %s toolchain %d %s", tc.name, tc.name, len(tc.name), tc.branch)
			if got := strings.TrimSpace(string(output)); got != want {
				t.Fatalf("compiler identity = %q, want %q", got, want)
			}
		})
	}
}

// The cache regression needs only the re-exporting package. Exercise the much
// larger go/build and go/importer dependency graph once under a non-gc runtime
// identity, where accidentally selecting a backend from runtime.Compiler fails.
func TestRuntimeCompilerToolchainPackages(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	conf := NewDefaultConf(ModeBuild)
	conf.BuildParallelism = 2
	conf.OutFile = filepath.Join(t.TempDir(), "toolchain")
	if runtime.GOOS == "windows" {
		conf.OutFile += ".exe"
	}
	conf.Overlay = runtimeCompilerOverlay(t, root, "llgo-cache")
	_, err = Build(Invocation{
		Args: []string{"."}, Config: conf,
		Dir: filepath.Join(root, "internal", "build", "testdata", "runtimecompiler", "toolchain"),
	})
	if err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(conf.OutFile).CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "llgo-cache" {
		t.Fatalf("toolchain packages: %v\n%s", err, output)
	}
}

func runtimeCompilerOverlay(t *testing.T, root, name string) map[string][]byte {
	t.Helper()
	patch := filepath.Join(root, "runtime", "_patch", "runtime", "compiler.go")
	source, err := os.ReadFile(patch)
	if err != nil {
		t.Fatal(err)
	}
	const declaration = `const Compiler = "llgo"`
	if strings.Count(string(source), declaration) != 1 {
		t.Fatal("runtime patch must contain exactly one Compiler declaration")
	}
	return map[string][]byte{patch: []byte(strings.Replace(string(source), declaration, `const Compiler = "`+name+`"`, 1))}
}
