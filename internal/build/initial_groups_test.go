//go:build !llgo

package build

import (
	"bytes"
	"encoding/json"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"weak"

	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	xpackages "golang.org/x/tools/go/packages"
	gossa "golang.org/x/tools/go/ssa"
)

func TestGroupInitialTestsByMemoryProfile(t *testing.T) {
	pprofTypes := types.NewPackage("runtime/pprof", "pprof")
	testdepsTypes := types.NewPackage("testing/internal/testdeps", "testdeps")
	testdepsTypes.SetImports([]*types.Package{pprofTypes})
	plainTypes := types.NewPackage("example.com/plain.test", "main")
	plainTypes.SetImports([]*types.Package{testdepsTypes})
	profiledTypes := types.NewPackage("example.com/profiled.test", "main")
	profiledTypes.SetImports([]*types.Package{testdepsTypes, pprofTypes})

	ssaProg := gossa.NewProgram(token.NewFileSet(), 0)
	for _, pkg := range []*types.Package{pprofTypes, testdepsTypes, plainTypes, profiledTypes} {
		ssaProg.CreatePackage(pkg, nil, nil, true)
	}
	pprof := &packages.Package{Types: pprofTypes}
	testdeps := &packages.Package{Types: testdepsTypes, Imports: map[string]*packages.Package{"runtime/pprof": pprof}}
	plain := &packages.Package{Types: plainTypes, Imports: map[string]*packages.Package{"testing/internal/testdeps": testdeps}}
	profiled := &packages.Package{Types: profiledTypes, Imports: map[string]*packages.Package{
		"testing/internal/testdeps": testdeps,
		"runtime/pprof":             pprof,
	}}
	prog := llssa.NewProgram(&llssa.Target{GOARCH: "amd64"})
	defer prog.Dispose()
	ctx := &context{
		prog: prog, progSSA: ssaProg, mode: ModeTest,
		buildConf: NewDefaultConf(ModeTest), initial: []*packages.Package{plain, profiled},
	}
	groups := groupInitialBuildsWithProfiles(ctx, nil, true)
	if len(groups) != 2 || groups[0].features.memoryProfile || !groups[1].features.memoryProfile ||
		!slices.Equal(groups[0].pkgs, []*packages.Package{plain}) ||
		!slices.Equal(groups[1].pkgs, []*packages.Package{profiled}) {
		t.Fatalf("profiled and ordinary tests shared a build group: %+v", groups)
	}
	ctx.buildConf.RunArgs = []string{"-test.memprofile=heap.out"}
	if !initialUsesMemoryProfile(ctx, plain) {
		t.Fatal("explicit -test.memprofile did not enable profiling")
	}
	ctx.buildConf.RunArgs = nil
	ctx.buildConf.CompileOnly = true
	if !initialUsesMemoryProfile(ctx, plain) {
		t.Fatal("go test -c binary cannot accept a later -test.memprofile flag")
	}
}

func TestGroupInitialBuilds(t *testing.T) {
	prog := llssa.NewProgram(&llssa.Target{GOARCH: "amd64"})
	defer prog.Dispose()
	plain := &packages.Package{Types: types.NewPackage("example.com/plain", "main")}
	local := &packages.Package{Types: types.NewPackage("example.com/local", "main")}
	other := &packages.Package{Types: types.NewPackage("example.com/other", "main")}
	prog.DeclareLocality(local.Types, "state", llssa.LocalityInfo{Locality: llssa.GoroutineLocal})
	ctx := &context{prog: prog, initial: []*packages.Package{plain, local, other}, mode: ModeBuild}
	groups := groupInitialBuilds(ctx, nil)
	if len(groups) != 2 || groups[0].features.localContext || !groups[1].features.localContext ||
		!slices.Equal(groups[0].pkgs, []*packages.Package{plain, other}) || !slices.Equal(groups[1].pkgs, []*packages.Package{local}) {
		t.Fatalf("incorrect per-initial groups: %+v", groups)
	}
	ctx.initial = []*packages.Package{plain, other}
	if groups := groupInitialBuilds(ctx, nil); len(groups) != 1 || len(groups[0].pkgs) != 2 {
		t.Fatalf("equivalent programs lost shared compilation: %+v", groups)
	}
	ctx.mode = ModeGen
	if groups := groupInitialBuilds(ctx, nil); groups != nil {
		t.Fatalf("ModeGen unexpectedly grouped: %+v", groups)
	}
	ctx.mode, ctx.initial = ModeBuild, []*packages.Package{plain}
	if groups := groupInitialBuilds(ctx, nil); groups != nil {
		t.Fatalf("single initial unexpectedly grouped: %+v", groups)
	}
}

func TestGroupInitialWasmFeatures(t *testing.T) {
	for _, provider := range []string{"wasi", "gojs", "emscripten"} {
		t.Run(provider, func(t *testing.T) {
			for _, source := range []string{
				`package main; import ("reflect"; "runtime"); func main() { reflect.ValueOf(func() {}).Call(nil); _ = runtime.FuncForPC(0) }`,
				`package library; import ("reflect"; "runtime"); func Use() { reflect.ValueOf(func() {}).Call(nil); _ = runtime.FuncForPC(0) }`,
			} {
				pkg := buildWasmReflectTestProgram(t, source)
				prog := llssa.NewProgram(&llssa.Target{GOARCH: "wasm", WasmProvider: provider})
				ctx := &context{prog: prog, progSSA: pkg.Prog, buildConf: &Config{BuildMode: BuildModeExe}, initial: []*packages.Package{
					{Types: types.NewPackage("example.com/plain", "plain")}, {Types: pkg.Pkg},
				}}
				groups := groupInitialBuilds(ctx, nil)
				if len(groups) != 2 || groups[0].features.funcInfoEntries || groups[0].features.reflectBridges ||
					!groups[1].features.funcInfoEntries || groups[1].features.reflectBridges != (provider == "wasi") {
					t.Fatalf("Wasm features leaked across initial programs for %s: %+v", pkg.Pkg.Name(), groups)
				}
				prog.Dispose()
			}
		})
	}
}

func TestMultiBuildRuntimeIsolationAndCache(t *testing.T) {
	t.Setenv(llgoBuildCache, "1")
	cacheDir := t.TempDir()
	oldCacheRoot := cacheRootFunc
	cacheRootFunc = func() string { return cacheDir }
	defer func() { cacheRootFunc = oldCacheRoot }()
	names := []string{"cprintf", "println", "fmtprintf"}
	baselines := make(map[string][]byte)
	fingerprints := make(map[string]string)
	build := func(names []string, warm bool) {
		t.Helper()
		conf := NewDefaultConf(ModeBuild)
		if runtime.GOOS == "windows" {
			// Build resolves a private Config copy; choose the expected suffix
			// explicitly instead of reading unresolved conf.AppExt afterward.
			conf.AppExt = ".exe"
		}
		conf.OutFile = t.TempDir() + string(os.PathSeparator)
		args := make([]string, len(names))
		for i, name := range names {
			args[i] = "../../benchmark/binary_size/" + name
		}
		pkgs, err := Do(args, conf)
		if err != nil {
			t.Fatal(err)
		}
		hits := 0
		for _, pkg := range pkgs {
			if pkg.CacheHit {
				hits++
			}
			for _, name := range names {
				if pkg.PkgPath != "github.com/xgo-dev/llgo/benchmark/binary_size/"+name {
					continue
				}
				if before, ok := fingerprints[name]; ok && before != pkg.Fingerprint {
					t.Errorf("%s fingerprint changed with unrelated initials", name)
				}
				fingerprints[name] = pkg.Fingerprint
				if name == "cprintf" && pkg.NeedRt {
					t.Error("cprintf unexpectedly requires runtime")
				}
			}
		}
		if warm && hits == 0 {
			t.Fatal("warm build did not reuse package archives")
		}
		for _, name := range names {
			output := filepath.Join(conf.OutFile, name+conf.AppExt)
			raw, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			if before, ok := baselines[name]; ok {
				// Windows PE headers may contain link timestamps. Still require
				// the same size, package fingerprint, and working program there.
				if len(before) != len(raw) || (runtime.GOOS != "windows" && !bytes.Equal(before, raw)) {
					t.Errorf("%s artifact changed with unrelated initials/cache: %d -> %d bytes", name, len(before), len(raw))
				}
			} else {
				baselines[name] = raw
			}
			assertBuiltProgram(t, output, "Hello, world")
		}
	}
	for _, name := range names {
		build([]string{name}, false)
	}
	build(names, true)
	slices.Reverse(names)
	build(names, true)
	build([]string{"cprintf"}, true)
	// Also start with a cold grouped build, then consume those archives from
	// standalone builds: cache population order must not affect the result.
	cacheDir = t.TempDir()
	// Compare artifact bytes within each cache population. Independent cold
	// builds can differ in dependency-init call ordering and, on ELF, the
	// resulting layout/alignment size even for standalone fmtprintf.
	// Reproducible cold linking is a separate concern; package fingerprints,
	// runtime requirements and execution remain checked across populations.
	baselines = make(map[string][]byte)
	build(names, false)
	for _, name := range names {
		build([]string{name}, true)
	}
}

func TestMixedInitialGroupsOutputAndErrors(t *testing.T) {
	root := writeMultiBuildModule(t, map[string]string{
		"cmd/plain/main.go": `package main; func main() { println("plain") }`,
		"cmd/local/main.go": `package main; import "fmt"; func main() { fmt.Println("local") }`,
		"cmd/broken/main.go": `package main
import ("fmt"; _ "unsafe")
//go:linkname missing C.llgo_initial_group_missing_symbol
func missing()
func main() { fmt.Println("broken"); missing() }
`,
	})
	conf := multiBuildConfig()
	if runtime.GOOS == "windows" {
		conf.AppExt = ".exe"
	}
	// Both names already belong to the user. A check-only grouped build must
	// neither try to link over the directory nor replace/delete the file.
	existingDir := filepath.Join(root, "plain"+conf.AppExt)
	existingFile := filepath.Join(root, "local"+conf.AppExt)
	if err := os.Mkdir(existingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const sentinel = "existing user file"
	if err := os.WriteFile(existingFile, []byte(sentinel), 0o644); err != nil {
		t.Fatal(err)
	}
	conf.BuildTrace = filepath.Join(t.TempDir(), "groups.json")
	if _, err := Build(Invocation{Args: []string{"./cmd/plain", "./cmd/local"}, Config: conf, Dir: root}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(conf.BuildTrace)
	if err != nil {
		t.Fatal(err)
	}
	var events []buildTraceEvent
	if err := json.Unmarshal(raw, &events); err != nil {
		t.Fatal(err)
	}
	builds := 0
	for _, event := range events {
		if event.Name == "build" && event.Phase == "X" {
			builds++
		}
	}
	if builds != 3 {
		t.Fatalf("expected parent and two group builds in shared trace, got %d", builds)
	}
	conf.BuildTrace = ""
	if info, err := os.Stat(existingDir); err != nil || !info.IsDir() {
		t.Fatalf("check-only build changed existing directory: %v", err)
	}
	if data, err := os.ReadFile(existingFile); err != nil || string(data) != sentinel {
		t.Fatalf("check-only build changed existing file: %q, %v", data, err)
	}
	conf.OutFile = t.TempDir() + string(os.PathSeparator)
	_, err = Build(Invocation{Args: []string{"./cmd/broken", "./cmd/plain", "./cmd/local"}, Config: conf, Dir: root})
	if err == nil || !strings.Contains(err.Error(), "broken") {
		t.Fatalf("group link failure was lost: %v", err)
	}
	for _, name := range []string{"plain", "local"} {
		assertBuiltProgram(t, filepath.Join(conf.OutFile, name+conf.AppExt), name)
	}
}

func TestInitialGroupUsesLoadablePackagePath(t *testing.T) {
	t.Run("test package", func(t *testing.T) {
		root := writeMultiBuildModule(t, map[string]string{
			"first/first_test.go": `package first; import "testing"; func TestFirst(t *testing.T) {}`,
		})
		conf := NewDefaultConf(ModeTest)
		ctx := &context{mode: ModeTest, buildConf: conf, commands: commandEnv{dir: root}}
		groups := []initialBuildGroup{{pkgs: []*packages.Package{{PkgPath: "example.com/multibuild/first.test"}}}}
		if _, err := buildInitialGroups(initialGroupInvocations(Invocation{}, ctx, groups)); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("GOPATH package", func(t *testing.T) {
		root := t.TempDir()
		pkgDir := filepath.Join(root, "first")
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pkgDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		conf := NewDefaultConf(ModeBuild)
		ctx := &context{mode: ModeBuild, buildConf: conf, commands: commandEnv{dir: root}}
		groups := []initialBuildGroup{{pkgs: []*packages.Package{{PkgPath: "_/not/loadable", Dir: pkgDir}}}}
		children := initialGroupInvocations(Invocation{}, ctx, groups)
		if !slices.Equal(children[0].Args, []string{"./first"}) {
			t.Fatalf("GOPATH child args = %q", children[0].Args)
		}
		current := []initialBuildGroup{{pkgs: []*packages.Package{{PkgPath: "command-line-arguments", Dir: root}}}}
		if args := initialGroupInvocations(Invocation{}, ctx, current)[0].Args; !slices.Equal(args, []string{"."}) {
			t.Fatalf("current-directory child args = %q", args)
		}
		env := append(os.Environ(), "GO111MODULE=off")
		loaded, err := xpackages.Load(&xpackages.Config{Mode: xpackages.NeedName, Dir: root, Env: env}, children[0].Args...)
		if err != nil {
			t.Fatal(err)
		}
		if xpackages.PrintErrors(loaded) != 0 || len(loaded) != 1 || loaded[0].Name != "main" {
			t.Fatalf("failed to reload GOPATH child: %+v", loaded)
		}
	})
}

func TestInitialGroupCoverageProfile(t *testing.T) {
	root := writeMultiBuildModule(t, map[string]string{
		"first/first.go":        "package first\nfunc Covered() int { return 1 }\n",
		"first/first_test.go":   "package first\nimport \"testing\"\nfunc TestCovered(t *testing.T) { Covered() }\n",
		"second/second.go":      "package second\nfunc Covered() int { return 2 }\n",
		"second/second_test.go": "package second\nimport \"testing\"\nfunc TestCovered(t *testing.T) { Covered() }\n",
	})
	profile := filepath.Join(t.TempDir(), "cover.out")
	conf := NewDefaultConf(ModeTest)
	conf.Coverage = &CoverageConfig{Mode: "count", Profile: profile}
	parent, err := newCoverageBuild(conf, commandEnv{dir: root})
	if err != nil {
		t.Fatal(err)
	}
	parent.close()
	ctx := &context{mode: ModeTest, buildConf: conf, commands: commandEnv{dir: root}}
	groups := []initialBuildGroup{
		{pkgs: []*packages.Package{{PkgPath: "example.com/multibuild/first.test"}}},
		{pkgs: []*packages.Package{{PkgPath: "example.com/multibuild/second.test"}}},
	}
	if _, err := buildInitialGroups(initialGroupInvocations(Invocation{}, ctx, groups)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Count(text, "mode: count\n") != 1 ||
		!strings.Contains(text, "example.com/multibuild/first/first.go:") ||
		!strings.Contains(text, "example.com/multibuild/second/second.go:") {
		t.Fatalf("grouped coverage profile lost a child fragment:\n%s", text)
	}
}

func TestInitialGroupInvocationsDoNotRetainParentPackages(t *testing.T) {
	children, parent := func() ([]Invocation, weak.Pointer[packages.Package]) {
		pkg := &packages.Package{PkgPath: "example.com/first.test"}
		conf := NewDefaultConf(ModeTest)
		conf.BuildParallelism = 3
		conf.Coverage = &CoverageConfig{Profile: "cover.out"}
		conf.coverage = &coverageBuild{noTests: []*packages.Package{pkg}}
		conf.coverageProfileInitialized = true
		ctx := &context{mode: ModeTest, buildConf: conf, commands: commandEnv{dir: "source"}}
		groups := []initialBuildGroup{{features: initialBuildFeatures{localContext: true}, pkgs: []*packages.Package{pkg}}}
		children := initialGroupInvocations(Invocation{disableMultiFallback: true}, ctx, groups)
		// Mutating the original input must not change a prepared invocation.
		groups[0].features.localContext = false
		pkg.PkgPath = "changed"
		conf.BuildParallelism = 7
		conf.Coverage.Profile = "changed"
		return children, weak.Make(pkg)
	}()
	child := children[0]
	if !slices.Equal(child.Args, []string{"example.com/first"}) || child.Dir != "source" ||
		child.Config.BuildParallelism != 3 || !child.initialFeatures.localContext ||
		!child.disableMultiFallback || !child.multipleInitials ||
		child.Config.Coverage.Profile != "cover.out" || child.Config.coverage != nil ||
		!child.Config.coverageProfileInitialized {
		t.Fatalf("group invocation lost its independent snapshot: %+v", child)
	}
	for range 10 {
		if parent.Value() == nil {
			break
		}
		runtime.GC()
	}
	if parent.Value() != nil {
		t.Fatal("prepared child still retains the parent package graph")
	}
	runtime.KeepAlive(children)
}
