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
	rateOnly := memoryProfileProviderIR(t, `package main
import "runtime"
func main() { runtime.MemProfileRate = 4096; println(runtime.MemProfileRate) }
`)
	if strings.Contains(rateOnly.allocator, "recordMemProfileAlloc") || hasMemProfileHookInstall(rateOnly.publicRuntime) {
		t.Fatal("setting MemProfileRate without reading a heap profile enabled sampling")
	}
	cpuOnly := memoryProfileProviderIR(t, `package main
import (
	"io"
	"runtime/pprof"
)
func main() { _ = pprof.StartCPUProfile(io.Discard); pprof.StopCPUProfile() }
`)
	if strings.Contains(cpuOnly.allocator, "recordMemProfileAlloc") || hasMemProfileHookInstall(cpuOnly.publicRuntime) {
		t.Fatal("CPU-only pprof use enabled heap sampling")
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
	heapProfile := memoryProfileProviderIR(t, `package main
import (
	"io"
	"runtime/pprof"
)
func main() { _ = pprof.WriteHeapProfile(io.Discard) }
`)
	if !strings.Contains(heapProfile.allocator, "recordMemProfileAlloc") || !hasMemProfileHookInstall(heapProfile.publicRuntime) {
		t.Fatal("runtime/pprof heap reader lost sampling")
	}
	httpProfile := memoryProfileProviderIR(t, `package main
import _ "net/http/pprof"
func main() {}
`)
	if !strings.Contains(httpProfile.allocator, "recordMemProfileAlloc") || !hasMemProfileHookInstall(httpProfile.publicRuntime) {
		t.Fatal("net/http/pprof heap endpoint lost sampling")
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

func TestMemoryProfileTestFlags(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{args: []string{"-test.run=TestOnly"}},
		{args: []string{"-test.memprofile="}},
		{args: []string{"-test.memprofilerate=0"}},
		{args: []string{"-test.memprofile=heap.out"}, want: true},
		{args: []string{"-test.memprofilerate=1"}, want: true},
	} {
		if got := testMemoryProfileRequested(tc.args); got != tc.want {
			t.Errorf("testMemoryProfileRequested(%q) = %v, want %v", tc.args, got, tc.want)
		}
	}
	if !testMemoryProfileRequired(ModeTest, &Config{CompileOnly: true}) {
		t.Fatal("compiled test binary lost support for later -test.memprofile")
	}
	if testMemoryProfileRequired(ModeBuild, &Config{CompileOnly: true}) {
		t.Fatal("compile-only non-test build enabled profiling")
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
		marker := `@"` + llssa.PkgRuntime + `.AllocU"(`
		start := -1
		for line := range strings.SplitSeq(ir, "\n") {
			if strings.HasPrefix(line, "define ") && strings.Contains(line, marker) {
				start = strings.Index(ir, line)
				break
			}
		}
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
		t.Fatal("runtime AllocU module was not observed")
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
