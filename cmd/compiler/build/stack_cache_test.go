package build

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	"github.com/xgo-dev/llvm"
)

func TestPthreadStackSizeRuntimeDependencyFingerprint(t *testing.T) {
	fingerprints := make(map[int64]string)
	for _, size := range []int64{0, 256 << 10} {
		dep := &aPackage{Package: &packages.Package{ID: llvmgen.PkgRuntime, PkgPath: llvmgen.PkgRuntime}}
		pkg := &aPackage{Package: &packages.Package{ID: "example/caller", PkgPath: "example/caller", Imports: map[string]*packages.Package{dep.PkgPath: dep.Package}}}
		ctx := &context{conf: &packages.Config{}, buildConf: &Config{PthreadStackSize: size}, pkgByID: map[string]*aPackage{dep.ID: dep, pkg.ID: pkg}, llvmVersion: "test"}
		if err := ctx.collectFingerprint(pkg); err != nil {
			t.Fatal(err)
		}
		data, err := decodeManifest(pkg.Manifest)
		if err != nil {
			t.Fatal(err)
		}
		if data.Package.PthreadStackSize != 0 {
			t.Fatal("ordinary caller got a direct stack input")
		}
		if len(data.Deps) != 1 || data.Deps[0].Fingerprint != dep.Fingerprint {
			t.Fatal("missing runtime dependency fingerprint")
		}
		fingerprints[size] = pkg.Fingerprint
	}
	if fingerprints[0] == fingerprints[256<<10] {
		t.Fatal("direct runtime importer failed to invalidate conservatively")
	}
}

// checkStackCacheNewProc verifies the configuration-independent caller ABI
// before TransformModule can inline NewProc or eliminate its declaration.
func checkStackCacheNewProc(t *testing.T, pkg Package) bool {
	t.Helper()
	fn := pkg.LPkg.Module().NamedFunction(llvmgen.PkgRuntime + ".NewProc")
	if fn.IsNil() {
		t.Errorf("%s did not declare NewProc", pkg.PkgPath)
		return false
	}
	if typ := fn.GlobalValueType(); typ.ParamTypesCount() != 2 || typ.IsFunctionVarArg() {
		t.Errorf("%s NewProc ABI = %s, want exactly two parameters", pkg.PkgPath, typ.String())
	}
	calls := 0
	for use := fn.FirstUse(); !use.IsNil(); use = use.NextUse() {
		call := use.User().IsACallInst()
		if call.IsNil() || call.CalledValue() != fn {
			continue
		}
		calls++
		if call.CalledFunctionType().ParamTypesCount() != 2 || call.OperandsCount() != 3 {
			t.Errorf("%s NewProc call still embeds stack configuration: %s", pkg.PkgPath, call.String())
		}
	}
	return calls > 0
}

func checkStackCacheRuntimeConstant(t *testing.T, pkg Package, size int64) {
	t.Helper()
	mod := pkg.LPkg.Module()
	global := mod.NamedGlobal(llvmgen.RuntimeGoroutineStackSizeVar)
	if global.IsNil() {
		t.Error("runtime did not emit goroutineStackSize")
		return
	}
	if !global.IsGlobalConstant() {
		t.Error("runtime goroutineStackSize is not an LLVM constant")
	}
	init := global.Initializer()
	if init.IsNil() || init.IsAConstantInt().IsNil() {
		t.Errorf("runtime goroutineStackSize has no integer initializer: %s", global.String())
	} else if got := init.ZExtValue(); got != uint64(size) {
		t.Errorf("runtime goroutineStackSize = %d, want %d", got, size)
	}
	fn := mod.NamedFunction(llvmgen.PkgRuntime + ".NewProc")
	if fn.IsNil() || fn.IsDeclaration() {
		t.Error("runtime did not define NewProc")
		return
	}
	backendCalls := 0
	for block := fn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			call := inst.IsACallInst()
			if call.IsNil() || call.CalledValue().Name() != llvmgen.PkgRuntime+".newprocBackend" {
				continue
			}
			backendCalls++
			if call.CalledFunctionType().ParamTypesCount() != 4 {
				t.Errorf("unexpected scheduler backend ABI: %s", call.String())
				continue
			}
			// The backend's stack argument must come from the configured
			// runtime global (or its already-folded constant), not an ABI arg.
			stack := call.Operand(2)
			if integer := stack.IsAConstantInt(); !integer.IsNil() {
				if integer.ZExtValue() != uint64(size) {
					t.Errorf("NewProc passes stack size %d, want %d", integer.ZExtValue(), size)
				}
			} else if load := stack.IsALoadInst(); load.IsNil() || load.Operand(0) != global {
				t.Errorf("NewProc stack size does not use its runtime constant: %s", stack.String())
			}
		}
	}
	if backendCalls != 1 {
		t.Errorf("NewProc calls scheduler backend %d times, want once", backendCalls)
	}
}

func TestPthreadStackSizePackageCache(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", repoRoot)
	t.Setenv(llgoBuildCache, "1")
	oldCacheRoot := cacheRootFunc
	cacheDir := t.TempDir()
	cacheRootFunc = func() string { return cacheDir }
	defer func() { cacheRootFunc = oldCacheRoot }()
	const fixturePath = "github.com/xgo-dev/llgo/cmd/compiler/build/testdata/stackcache/"
	seenSize := make(map[int64]bool)
	for phase, tc := range []struct {
		size  int64
		force bool
	}{
		{0, false}, {256 << 10, false}, {512 << 10, false}, {0, false}, {256 << 10, true}, {-1, false},
	} {
		t.Run(fmt.Sprintf("%d-size%d-force%v", phase, tc.size, tc.force), func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.PthreadStackSize = tc.size
			conf.ForceRebuild = tc.force
			conf.OutFile = filepath.Join(t.TempDir(), "stackcache")
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			conf.BuildParallelism = 2
			normalizedSize := max(0, tc.size)
			generated := make(map[string]bool)
			runtimeChecked := false
			var hookMu sync.Mutex
			conf.ModuleHook = func(pkg Package) {
				if pkg.CacheHit || pkg.LPkg == nil {
					return
				}
				hookMu.Lock()
				defer hookMu.Unlock()
				switch pkg.PkgPath {
				case llvmgen.PkgRuntime:
					checkStackCacheNewProc(t, pkg)
					checkStackCacheRuntimeConstant(t, pkg, normalizedSize)
					runtimeChecked = true
				case fixturePath + "spawn", fixturePath + "instance":
					generated[pkg.PkgPath] = checkStackCacheNewProc(t, pkg)
				}
			}
			pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: filepath.Join(repoRoot, "internal", "build", "testdata", "stackcache")})
			if err != nil {
				t.Fatal(err)
			}
			found := make(map[string]bool)
			for _, pkg := range pkgs {
				if pkg.PkgPath == llvmgen.PkgRuntime {
					found["runtime"] = true
					wantHit := seenSize[normalizedSize] && !tc.force
					if pkg.CacheHit != wantHit {
						t.Errorf("runtime CacheHit = %v, want %v", pkg.CacheHit, wantHit)
					}
					if !wantHit && !runtimeChecked {
						t.Error("runtime stack constant was not checked on a cache miss")
					}
					continue
				}
				if pkg.PkgPath == strings.TrimSuffix(fixturePath, "/") {
					found["main"] = true
					if pkg.CacheHit {
						t.Error("main package must not use the package cache")
					}
					continue
				}
				name, ok := strings.CutPrefix(pkg.PkgPath, fixturePath)
				if !ok {
					continue
				}
				found[name] = true
				// Neither ordinary nor instantiated generic go statements
				// embed the stack size. They share the plain package's hits.
				wantHit := phase > 0 && !tc.force
				if pkg.CacheHit != wantHit {
					t.Errorf("%s CacheHit = %v, want %v", name, pkg.CacheHit, wantHit)
				}
				if !wantHit && (name == "spawn" || name == "instance") && !generated[pkg.PkgPath] {
					t.Errorf("%s did not emit a two-argument goroutine entry call", name)
				}
			}
			for _, name := range []string{"main", "runtime", "plain", "spawn", "generic", "instance"} {
				if !found[name] {
					t.Errorf("missing package %s", name)
				}
			}
			if out, err := exec.Command(conf.OutFile).CombinedOutput(); err != nil {
				t.Fatalf("run: %v\n%s", err, out)
			}
			seenSize[normalizedSize] = true
		})
	}
}
