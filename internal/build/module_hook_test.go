//go:build !llgo
// +build !llgo

package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestModuleHookReceivesMainPackageModule(t *testing.T) {
	conf := NewDefaultConf(ModeGen)

	counts := make(map[string]int)
	snapshots := make(map[string]string)
	conf.ModuleHook = func(pkg Package) {
		counts[pkg.PkgPath]++
		if _, ok := snapshots[pkg.PkgPath]; !ok {
			snapshots[pkg.PkgPath] = pkg.LPkg.String()
		}
	}

	pkgs, err := Do([]string{"../../cl/_testgo/print"}, conf)
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 initial package, got %d", len(pkgs))
	}

	mainPkg := pkgs[0].PkgPath
	if counts[mainPkg] != 1 {
		t.Fatalf("expected hook to fire once for %s, got %d", mainPkg, counts[mainPkg])
	}
	if snapshots[mainPkg] == "" {
		t.Fatalf("expected non-empty module snapshot for %s", mainPkg)
	}
}

func TestMemoryProfileConsumerSelectsAllocatorInstrumentation(t *testing.T) {
	cacheDir := t.TempDir()
	oldCacheRoot := cacheRootFunc
	cacheRootFunc = func() string { return cacheDir }
	defer func() { cacheRootFunc = oldCacheRoot }()

	plain := memoryProfileProviderIR(t, `package main
import "runtime"
func main() { println(runtime.GOOS) }
`)
	if strings.Contains(plain.allocator, "recordMemProfileAlloc") {
		t.Fatalf("plain executable allocator retained memory profiling:\n%s", plain.allocator)
	}
	if hasMemProfileHookInstall(plain.publicRuntime) {
		t.Fatalf("plain executable runtime installed memory-profile hooks:\n%s", plain.publicRuntime)
	}

	profiled := memoryProfileProviderIR(t, `package main
import "runtime"
func main() { runtime.MemProfile(nil, false) }
`)
	if !strings.Contains(profiled.allocator, "recordMemProfileAlloc") {
		t.Fatalf("memory-profile consumer allocator lost recording:\n%s", profiled.allocator)
	}
	if !hasMemProfileHookInstall(profiled.publicRuntime) {
		t.Fatalf("memory-profile consumer runtime lost hook installation:\n%s", profiled.publicRuntime)
	}
}

func TestMemoryProfileLibraryModeSelection(t *testing.T) {
	for _, mode := range []BuildMode{BuildModeCArchive, BuildModeCShared} {
		if !enableMemoryProfiling(mode, "") {
			t.Errorf("%s build disabled externally callable memory profiling", mode)
		}
	}
	if enableMemoryProfiling(BuildModeExe, "") {
		t.Error("plain executable enabled memory profiling")
	}
	if !enableMemoryProfiling(BuildModeExe, "runtime/pprof") {
		t.Error("profile consumer did not enable executable memory profiling")
	}
}

type memoryProfileProviders struct {
	allocator     string
	publicRuntime string
}

func memoryProfileProviderIR(t *testing.T, source string) memoryProfileProviders {
	t.Helper()
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainFile, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	conf := NewDefaultConf(ModeGen)
	var providers memoryProfileProviders
	conf.ModuleHook = func(pkg Package) {
		if pkg.PkgPath == "runtime" || pkg.PkgPath == altPkgPathPrefix+"runtime" {
			providers.publicRuntime = pkg.LPkg.String()
		}
		if pkg.PkgPath != llssa.PkgRuntime {
			return
		}
		ir := pkg.LPkg.String()
		marker := `define ptr @"` + llssa.PkgRuntime + `.AllocZ"(`
		start := strings.Index(ir, marker)
		if start < 0 {
			return
		}
		end := strings.Index(ir[start:], "\n}")
		if end >= 0 {
			providers.allocator = ir[start : start+end+2]
		}
	}
	pkgs, err := Do([]string{mainFile}, conf)
	if err != nil {
		t.Fatalf("generate memory-profile allocator IR: %v", err)
	}
	if len(pkgs) == 1 && pkgs[0].LPkg != nil {
		defer pkgs[0].LPkg.Prog.Dispose()
	}
	if providers.allocator == "" {
		t.Fatal("runtime AllocZ module was not observed")
	}
	if providers.publicRuntime == "" {
		t.Fatal("public runtime module was not observed")
	}
	return providers
}

func hasMemProfileHookInstall(ir string) bool {
	for line := range strings.SplitSeq(ir, "\n") {
		if strings.Contains(line, "call void") && strings.Contains(line, ".installMemProfileHooks(") {
			return true
		}
	}
	return false
}
