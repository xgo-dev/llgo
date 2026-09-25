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

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

// Rebuild the caller even when the dependency is cached. Its external function
// declarations must reflect source annotations, including annotation-only edits. The bridge package also verifies transitive
// caller cache invalidation.
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
		{"cache-disabled", true, 0, ""},
	} {
		t.Run(phase.name, func(t *testing.T) {
			if phase.name == "cache-disabled" {
				t.Setenv(llgoBuildCache, "0")
			}
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
			checkedRuntime := false
			conf.ModuleHook = func(pkg Package) {
				if pkg.LPkg == nil {
					return
				}
				mu.Lock()
				defer mu.Unlock()
				if pkg.PkgPath == ssa.PkgRuntime {
					checkRuntimeFunctionAttributes(t, pkg.LPkg.Module())
					checkedRuntime = true
					return
				}
				if pkg.PkgPath == prefix+"/dep" {
					fn := pkg.LPkg.Module().NamedFunction(ssa.PkgRuntime + ".Panic")
					if fn.IsNil() {
						t.Error("missing imported runtime.Panic declaration")
						return
					}
					for _, name := range []string{"cold", "noreturn"} {
						if fn.GetEnumFunctionAttribute(llvm.AttributeKindID(name)).IsNil() {
							t.Errorf("imported runtime.Panic lost %s", name)
						}
					}
				}
				if pkg.PkgPath != prefix && pkg.PkgPath != prefix+"/bridge" {
					return
				}
				symbols := []string{"Stop"}
				if pkg.PkgPath == prefix {
					symbols = append(symbols, "Generic[int]", "T.Stop")
				}
				for _, symbol := range symbols {
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
			if (phase.name == "first" || phase.name == "nogc") && !checkedRuntime {
				t.Fatal("runtime module was not checked")
			}
			found := 0
			for _, pkg := range pkgs {
				if pkg.PkgPath == prefix+"/dep" || pkg.PkgPath == prefix+"/bridge" {
					found++
					if phase.hit >= 0 && pkg.CacheHit != (phase.hit == 1) {
						t.Errorf("%s CacheHit = %v, want %v", pkg.PkgPath, pkg.CacheHit, phase.hit == 1)
					}
				}
			}
			if found != 2 {
				t.Fatal("missing dependency build result")
			}
			if out, err := exec.Command(conf.OutFile).CombinedOutput(); err != nil {
				t.Fatalf("panic/defer/recover execution: %v\n%s", err, out)
			}
		})
	}
}

func checkRuntimeFunctionAttributes(t *testing.T, mod llvm.Module) {
	t.Helper()
	for _, tc := range []struct {
		name           string
		cold, noreturn bool
	}{
		{"Panic", true, true}, {"PanicSignal", true, true},
		{"PanicErrorString", true, true}, {"PanicIndex", true, true}, {"PanicIndexU", true, true},
		{"PanicTypeAssertionError", true, true}, {"panicBounds", true, true},
		{"PanicSliceConvert", true, true}, {"PanicTypeAssert", true, true},
		{"panicmakeslicelen", true, true}, {"panicmakeslicecap", true, true},
		{"panicgrowslicelen", true, true}, {"panicMakeChanSize", true, true},
		{"panicSendOnClosedChan", true, true}, {"Goexit", false, true},
		{"TracePanic", true, false}, {"throw", true, false}, {"fatal", true, false},
		{"unreachableMethod", true, false},
		// These helpers also run on normal paths and may return.
		{"Rethrow", false, false}, {"AssertNilDeref", false, false},
		{"PanicWrapNilPointer", false, false},
	} {
		fn := mod.NamedFunction(ssa.PkgRuntime + "." + tc.name)
		if fn.IsNil() {
			t.Errorf("missing runtime.%s", tc.name)
			continue
		}
		for name, want := range map[string]bool{"cold": tc.cold, "noreturn": tc.noreturn} {
			if got := !fn.GetEnumFunctionAttribute(llvm.AttributeKindID(name)).IsNil(); got != want {
				t.Errorf("runtime.%s: %s = %v, want %v", tc.name, name, got, want)
			}
		}
		if tc.noreturn && !fn.GetEnumFunctionAttribute(llvm.AttributeKindID("nounwind")).IsNil() {
			t.Errorf("runtime.%s must preserve unwinding", tc.name)
		}
	}
}
