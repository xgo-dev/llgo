//go:build !llgo
// +build !llgo

package build

import (
	"bytes"
	"debug/macho"
	"fmt"
	"go/ast"
	gobuild "go/build"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/buildenv"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/meta"
	"github.com/xgo-dev/llgo/internal/mockable"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestMain(m *testing.M) {
	old := cacheRootFunc
	td, _ := os.MkdirTemp("", "llgo-cache-*")
	cacheRootFunc = func() string { return td }
	code := m.Run()
	cacheRootFunc = old
	_ = os.RemoveAll(td)
	os.Exit(code)
}

func TestConfigCloneDoesNotAliasInput(t *testing.T) {
	input := &Config{
		RunArgs:      []string{"run"},
		GoBuildFlags: []string{"-tags=custom"},
		Overlay:      map[string][]byte{"input.go": []byte("package input")},
		GlobalRewrites: map[string]Rewrites{
			"example.com/p": {"value": "input"},
			"nil":           nil,
		},
	}
	cloned := input.clone()
	cloned.RunArgs[0] = "changed"
	cloned.GoBuildFlags[0] = "-tags=changed"
	cloned.Overlay["input.go"][0] = 'P'
	cloned.GlobalRewrites["example.com/p"]["value"] = "changed"
	cloned.GlobalRewrites["new"] = Rewrites{"value": "new"}

	if got := input.RunArgs[0]; got != "run" {
		t.Fatalf("input RunArgs changed to %q", got)
	}
	if got := input.GoBuildFlags[0]; got != "-tags=custom" {
		t.Fatalf("input GoBuildFlags changed to %q", got)
	}
	if got := string(input.Overlay["input.go"]); got != "package input" {
		t.Fatalf("input overlay changed to %q", got)
	}
	if got := input.GlobalRewrites["example.com/p"]["value"]; got != "input" {
		t.Fatalf("input rewrite changed to %q", got)
	}
	if _, ok := input.GlobalRewrites["new"]; ok {
		t.Fatal("cloned rewrite map aliases input map")
	}
	if rewrites, ok := cloned.GlobalRewrites["nil"]; !ok || rewrites != nil {
		t.Fatalf("nil rewrite entry was not preserved: %#v", rewrites)
	}
	if got := (*Config)(nil).clone(); got != nil {
		t.Fatalf("nil Config clone = %#v", got)
	}
}

func TestResolveBuildConfigDefaultsAndValidation(t *testing.T) {
	resolved, err := resolveBuildConfig(&Config{
		BuildMode:    BuildModeCArchive,
		DeadcodeDrop: true,
		SizeReport:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.DeadcodeDrop {
		t.Fatal("non-executable build retained dead-code dropping")
	}
	if resolved.SizeFormat != "text" || resolved.SizeLevel != "module" {
		t.Fatalf("size report defaults = %q, %q", resolved.SizeFormat, resolved.SizeLevel)
	}
	if _, err := resolveBuildConfig(&Config{SizeReport: true, SizeLevel: "invalid"}); err == nil {
		t.Fatal("invalid size-reporting level succeeded")
	}
	if _, err := resolveBuildConfig(nil); err == nil {
		t.Fatal("nil build config succeeded")
	}
}

func TestNewDefaultConfDoesNotCreateBinDir(t *testing.T) {
	binDir := filepath.Join(t.TempDir(), "not-created", "bin")
	t.Setenv("GOBIN", binDir)
	conf := NewDefaultConf(ModeBuild)
	if conf.BinPath != binDir {
		t.Fatalf("BinPath = %q, want %q", conf.BinPath, binDir)
	}
	if _, err := os.Stat(binDir); !os.IsNotExist(err) {
		t.Fatalf("NewDefaultConf created bin directory: %v", err)
	}
}

func TestDoDoesNotModifyConfigOnValidationError(t *testing.T) {
	input := &Config{
		RunArgs: []string{"arg"},
		GlobalRewrites: map[string]Rewrites{
			"example.com/p": {"value": "input"},
		},
		LinkOptions: LinkOptions{DWARF: DWARFMode(255)},
	}
	before := input.clone()
	if _, err := Do(nil, input); err == nil {
		t.Fatal("Do() succeeded with invalid DWARF mode")
	}
	if !reflect.DeepEqual(input, before) {
		t.Fatalf("Do() modified input config:\n got: %#v\nwant: %#v", input, before)
	}
	if _, err := Do(nil, nil); err == nil {
		t.Fatal("Do() succeeded with nil config")
	}
}

func TestInvocationUsesExplicitWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/requestdir\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "requestdir.go"), []byte("package requestdir\n\nfunc F() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	conf := NewDefaultConf(ModeGen)
	t.Setenv(llgoBuildCache, "0")
	ambientPath := os.Getenv("PATH")
	pkgs, err := Build(Invocation{
		Args:   []string{"."},
		Config: conf,
		Dir:    dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].PkgPath != "example.com/requestdir" {
		t.Fatalf("Build returned packages = %+v, want example.com/requestdir", pkgs)
	}
	if got := os.Getenv("PATH"); got != ambientPath {
		t.Fatalf("Build changed process PATH from %q to %q", ambientPath, got)
	}
	pkgs[0].LPkg.Prog.Dispose()
}

func TestConcurrentInvocationsIsolateFrontendOptions(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/frontend\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "frontend.go"), []byte("package frontend\n\nfunc F(v int) int { return v + 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(llgoBuildCache, "0")

	type result struct {
		debug bool
		pkgs  []Package
		err   error
	}
	results := make(chan result, 2)
	for _, debug := range []bool{false, true} {
		conf := NewDefaultConf(ModeGen)
		conf.LinkOptions.DWARF = DWARFOmit
		if debug {
			conf.LinkOptions.DWARF = DWARFPreserve
			conf.Verbose = true
		}
		go func() {
			pkgs, err := Build(Invocation{
				Args:   []string{"."},
				Config: conf,
				Dir:    dir,
			})
			results <- result{debug: debug, pkgs: pkgs, err: err}
		}()
	}
	for range 2 {
		got := <-results
		if got.err != nil {
			t.Fatal(got.err)
		}
		if len(got.pkgs) != 1 || got.pkgs[0].LPkg == nil {
			t.Fatalf("Build returned packages = %+v, want one compiled package", got.pkgs)
		}
		t.Cleanup(got.pkgs[0].LPkg.Prog.Dispose)
		hasDebugInfo := strings.Contains(got.pkgs[0].LPkg.String(), "!llvm.dbg.cu")
		if hasDebugInfo != got.debug {
			t.Fatalf("debug=%v produced hasDebugInfo=%v", got.debug, hasDebugInfo)
		}
	}
}

func TestConcurrentDeadcodeBuildsUseIndependentWorkerContexts(t *testing.T) {
	if !buildenv.Dev {
		t.Skip("deadcode drop requires a development build")
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(repoRoot, "cl", "_testdrop", "direct_method")
	t.Setenv("LLGO_ROOT", repoRoot)
	t.Setenv(llgoBuildCache, "0")

	type result struct {
		output string
		err    error
	}
	results := make(chan result, 2)
	for i := range 2 {
		output := filepath.Join(t.TempDir(), fmt.Sprintf("direct-method-%d", i))
		conf := NewDefaultConf(ModeBuild)
		conf.DeadcodeDrop = true
		conf.PCLNMode = PCLNNone
		conf.BuildParallelism = 2
		conf.OutFile = output
		go func() {
			_, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: fixture})
			results <- result{output: output, err: err}
		}()
	}

	for range 2 {
		got := <-results
		if got.err != nil {
			t.Fatal(got.err)
		}
		data, err := exec.Command(got.output).CombinedOutput()
		if err != nil {
			t.Fatalf("run %s: %v\n%s", got.output, err, data)
		}
		if strings.TrimSpace(string(data)) != "42" {
			t.Fatalf("run %s output = %q, want 42", got.output, data)
		}
	}
}

func TestDeadcodeBuildColdAndHotPackageCache(t *testing.T) {
	if !buildenv.Dev {
		t.Skip("deadcode drop requires a development build")
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(repoRoot, "cl", "_testdrop", "direct_method")
	t.Setenv("LLGO_ROOT", repoRoot)
	t.Setenv(llgoBuildCache, "1")

	for _, parallelism := range []int{1, 8} {
		t.Run(fmt.Sprintf("p%d", parallelism), func(t *testing.T) {
			oldCacheRoot := cacheRootFunc
			cacheDir := t.TempDir()
			cacheRootFunc = func() string { return cacheDir }
			defer func() { cacheRootFunc = oldCacheRoot }()

			build := func(name string) []Package {
				output := filepath.Join(t.TempDir(), name)
				conf := NewDefaultConf(ModeBuild)
				conf.DeadcodeDrop = true
				conf.PCLNMode = PCLNNone
				conf.BuildParallelism = parallelism
				conf.OutFile = output
				pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: fixture})
				if err != nil {
					t.Fatal(err)
				}
				data, err := exec.Command(output).CombinedOutput()
				if err != nil {
					t.Fatalf("run %s: %v\n%s", output, err, data)
				}
				if strings.TrimSpace(string(data)) != "42" {
					t.Fatalf("run %s output = %q, want 42", output, data)
				}
				return pkgs
			}

			first := build("cold")
			for _, pkg := range first {
				if pkg.CacheHit {
					t.Fatalf("cold build unexpectedly hit package cache for %s", pkg.PkgPath)
				}
			}
			second := build("hot")
			cacheHits := 0
			for _, pkg := range second {
				if pkg.CacheHit {
					cacheHits++
				}
			}
			if cacheHits == 0 {
				t.Fatal("hot build did not reuse any package archives")
			}
		})
	}
}

func TestGenericLocalTypeColdAndHotPackageCache(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(repoRoot, "internal", "build", "testdata", "genericlocalcache")
	const (
		cacheRootEnv  = "LLGO_TEST_GENERIC_LOCAL_CACHE_ROOT"
		cachePhaseEnv = "LLGO_TEST_GENERIC_LOCAL_CACHE_PHASE"
	)
	if cacheRoot := os.Getenv(cacheRootEnv); cacheRoot != "" {
		cacheRootFunc = func() string { return cacheRoot }
		t.Setenv("LLGO_ROOT", repoRoot)
		t.Setenv(llgoBuildCache, "1")

		conf := NewDefaultConf(ModeTest)
		conf.OutFile = filepath.Join(t.TempDir(), os.Getenv(cachePhaseEnv))
		conf.RunArgs = []string{"-test.run=^TestLocalRuntimeType$"}
		pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: fixture})
		if err != nil {
			t.Fatal(err)
		}
		switch phase := os.Getenv(cachePhaseEnv); phase {
		case "cold":
			for _, pkg := range pkgs {
				if pkg.CacheHit {
					t.Fatalf("cold build unexpectedly hit package cache for %s", pkg.PkgPath)
				}
			}
		case "hot":
			for _, pkg := range pkgs {
				if pkg.CacheHit {
					return
				}
			}
			t.Fatal("hot build did not reuse any package archives")
		default:
			t.Fatalf("unknown cache phase %q", phase)
		}
		return
	}

	cacheRoot := t.TempDir()
	for _, phase := range []string{"cold", "hot"} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestGenericLocalTypeColdAndHotPackageCache$", "-test.count=1")
		cmd.Env = append(os.Environ(), cacheRootEnv+"="+cacheRoot, cachePhaseEnv+"="+phase)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s cache build: %v\n%s", phase, err, out)
		}
	}
}

func TestResolveOutputsUsesInvocationDirectory(t *testing.T) {
	dir := t.TempDir()
	out := &OutFmtDetails{
		Out: "app",
		Bin: filepath.Join("firmware", "app.bin"),
		Hex: filepath.Join(dir, "app.hex"),
	}
	resolveOutputs(dir, out)
	if out.Out != filepath.Join(dir, "app") {
		t.Fatalf("Out = %q", out.Out)
	}
	if out.Bin != filepath.Join(dir, "firmware", "app.bin") {
		t.Fatalf("Bin = %q", out.Bin)
	}
	if out.Hex != filepath.Join(dir, "app.hex") {
		t.Fatalf("absolute Hex changed to %q", out.Hex)
	}
}

func TestConfigureCommandUsesBuildSnapshot(t *testing.T) {
	dir := t.TempDir()
	commands := commandEnv{dir: dir, environ: []string{"BUILD_MARKER=before"}}
	cmd := commands.configure(exec.Command("unused"))
	commands.environ[0] = "BUILD_MARKER=after"
	if cmd.Dir != dir {
		t.Fatalf("command Dir = %q, want %q", cmd.Dir, dir)
	}
	if got, want := cmd.Env, []string{"BUILD_MARKER=before"}; !slices.Equal(got, want) {
		t.Fatalf("command Env = %q, want %q", got, want)
	}
}

func TestLinkObjFilesReportsOutputDirectoryError(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(parent, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := &context{buildConf: &Config{BuildMode: BuildModeExe}}
	if err := linkObjFiles(ctx, filepath.Join(parent, "app"), nil, nil, false); err == nil {
		t.Fatal("linkObjFiles succeeded below a regular file")
	}
}

func TestRewritePrebuiltFuncTabEligibilityAndDiagnostic(t *testing.T) {
	rewritePrebuiltFuncTab(nil, "missing", true)
	rewritePrebuiltFuncTab(&context{}, "missing", true)

	prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	ctx := &context{prog: prog, buildConf: &Config{
		BuildMode: BuildModeExe,
		Goos:      "linux",
		Goarch:    "amd64",
	}}
	rewritePrebuiltFuncTab(ctx, "missing", true) // sites disabled

	prog.EnableFuncInfoSites(true)
	ctx.buildConf.Target = "wasi"
	rewritePrebuiltFuncTab(ctx, "missing", true)
	ctx.buildConf.Target = ""
	ctx.buildConf.BuildMode = BuildModeCShared
	rewritePrebuiltFuncTab(ctx, "missing", true)
	ctx.buildConf.BuildMode = BuildModeExe

	t.Setenv("LLGO_PCLNPOST", "0")
	rewritePrebuiltFuncTab(ctx, "missing", true)
	t.Setenv("LLGO_PCLNPOST", "1")
	rewritePrebuiltFuncTab(ctx, "missing", false)

	stderr, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = stderr
	t.Cleanup(func() { os.Stderr = oldStderr })
	rewritePrebuiltFuncTab(ctx, filepath.Join(t.TempDir(), "missing"), true)
	os.Stderr = oldStderr
	if err := stderr.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(stderr.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte("prebuilt functab rewrite skipped")) {
		t.Fatalf("rewrite diagnostic = %q", got)
	}
}

func TestWithEnvLastValueWins(t *testing.T) {
	got := withEnv([]string{"A=old", "B=keep", "malformed", "A=older"}, "A=new", "C=value")
	want := []string{"B=keep", "A=new", "C=value"}
	if !slices.Equal(got, want) {
		t.Fatalf("withEnv = %q, want %q", got, want)
	}
}

func TestWithResolvedGoToolchain(t *testing.T) {
	tests := []struct {
		name      string
		environ   []string
		goversion string
		want      []string
	}{
		{"replace existing", []string{"PATH=/bin", "GOTOOLCHAIN=auto"}, "go1.25.0", []string{"PATH=/bin", "GOTOOLCHAIN=go1.25.0"}},
		{"append missing", []string{"PATH=/bin"}, "go1.25.0", []string{"PATH=/bin", "GOTOOLCHAIN=go1.25.0"}},
		{"preserve development version", []string{"PATH=/bin"}, "devel go1.26-deadbeef", []string{"PATH=/bin"}},
		{"preserve empty version", []string{"PATH=/bin"}, "", []string{"PATH=/bin"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := withResolvedGoToolchain(tt.environ, tt.goversion)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("withResolvedGoToolchain = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClosePackageMetas(t *testing.T) {
	b := meta.NewBuilder()
	b.Sym("pkg.main")
	written, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "pkg.meta")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := written.WriteTo(f); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	loaded, err := readMeta(path)
	if err != nil {
		t.Fatal(err)
	}
	pkg := &aPackage{Package: &packages.Package{}, Meta: loaded}
	ctx := &context{pkgs: map[*packages.Package]Package{pkg.Package: pkg}}
	ctx.closePackageMetas()
	if pkg.Meta != nil {
		t.Fatal("package metadata was not cleared")
	}
	if n, err := loaded.WriteTo(io.Discard); err != nil || n != 0 {
		t.Fatalf("closed metadata still has mapped bytes: n=%d err=%v", n, err)
	}
}

func TestNeedsLinuxNoPIE(t *testing.T) {
	ctx := &context{buildConf: &Config{Goos: "linux"}}
	if !needsLinuxNoPIE(ctx, nil) {
		t.Fatal("linux executable link should default to -no-pie")
	}
	for _, flag := range []string{"-pie", "-static-pie", "-no-pie", "-nopie"} {
		if needsLinuxNoPIE(ctx, []string{flag}) {
			t.Fatalf("explicit %s should not be overridden", flag)
		}
	}
	ctx.buildConf.Goos = "darwin"
	if needsLinuxNoPIE(ctx, nil) {
		t.Fatal("non-linux executable link should not force -no-pie")
	}
	ctx.buildConf.Goos = "linux"
	ctx.buildConf.Target = "wasi"
	if needsLinuxNoPIE(ctx, nil) {
		t.Fatal("named targets should not force host linux -no-pie")
	}
}

func TestDefaultBuildTags(t *testing.T) {
	const base = "llgo,math_big_pure_go,purego"
	for _, test := range []struct {
		name   string
		goarch string
		target string
		want   string
	}{
		{name: "native", goarch: "arm64", want: base},
		{name: "raw wasm", goarch: "wasm", want: base + ",nogc"},
		{name: "configured wasm target", goarch: "wasm", target: "wasip1", want: base},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := defaultBuildTags(test.goarch, test.target); got != test.want {
				t.Fatalf("defaultBuildTags(%q, %q) = %q, want %q", test.goarch, test.target, got, test.want)
			}
		})
	}
}

func TestWasmRuntimeAvoidsNativeHostDependencies(t *testing.T) {
	runtimeDir := filepath.Join(env.LLGoRuntimeDir(), "internal", "lib", "runtime")
	for _, goos := range []string{"js", "wasip1"} {
		t.Run(goos, func(t *testing.T) {
			ctx := gobuild.Default
			ctx.GOOS = goos
			ctx.GOARCH = "wasm"
			ctx.BuildTags = []string{"llgo", "nogc"}
			pkg, err := ctx.ImportDir(runtimeDir, 0)
			if err != nil {
				t.Fatal(err)
			}

			selected := make(map[string]bool)
			for _, name := range append(pkg.GoFiles, pkg.CgoFiles...) {
				selected[name] = true
				file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(runtimeDir, name), nil, parser.ImportsOnly)
				if err != nil {
					t.Fatal(err)
				}
				for _, spec := range file.Imports {
					path, err := strconv.Unquote(spec.Path.Value)
					if err != nil {
						t.Fatal(err)
					}
					switch path {
					case "github.com/xgo-dev/llgo/runtime/internal/clite/libuv",
						"github.com/xgo-dev/llgo/runtime/internal/clite/bdwgc":
						t.Fatalf("wasm selected %s, which imports native host dependency %s", name, path)
					}
				}
			}

			for _, name := range []string{
				"mfinal_nogc.go",
				"runtime_baremetal.go",
				"signal_baremetal_llgo.go",
				"time_wasm_llgo.go",
				"unwind_wasm_llgo.go",
			} {
				if !selected[name] {
					t.Errorf("wasm runtime did not select %s", name)
				}
			}
		})
	}
}

func TestBaremetalRuntimeAvoidsLocalityDirectives(t *testing.T) {
	for _, relative := range []string{
		filepath.Join("internal", "runtime"),
		filepath.Join("internal", "lib", "runtime"),
	} {
		t.Run(relative, func(t *testing.T) {
			dir := filepath.Join(env.LLGoRuntimeDir(), relative)
			ctx := gobuild.Default
			ctx.BuildTags = []string{"llgo", "baremetal"}
			pkg, err := ctx.ImportDir(dir, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range append(pkg.GoFiles, pkg.CgoFiles...) {
				content, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil {
					t.Fatal(err)
				}
				if bytes.Contains(content, []byte("//llgo:tls")) || bytes.Contains(content, []byte("//llgo:gls")) {
					t.Fatalf("bare-metal runtime selected locality directive in %s", name)
				}
			}
		})
	}
}

func TestNeedsLinuxExportDynamic(t *testing.T) {
	t.Setenv(llgoFuncInfo, "")
	ctx := &context{buildConf: &Config{Goos: "linux"}}
	if !needsLinuxExportDynamic(ctx) {
		t.Fatal("linux funcinfo executable should export dynamic symbols")
	}
	if got := linuxExportDynamicArgs(ctx); strings.Join(got, " ") != "-Wl,--export-dynamic-symbol=main.* -Wl,--export-dynamic-symbol=command-line-arguments.*" {
		t.Fatalf("linuxExportDynamicArgs = %v", got)
	}
	t.Setenv(llgoFuncInfo, "0")
	if needsLinuxExportDynamic(ctx) {
		t.Fatal("LLGO_FUNCINFO=0 should disable dynamic symbol export")
	}
	if got := linuxExportDynamicArgs(ctx); got != nil {
		t.Fatalf("disabled linuxExportDynamicArgs = %v, want nil", got)
	}
	t.Setenv(llgoFuncInfo, "1")
	ctx.buildConf.Goos = "darwin"
	if needsLinuxExportDynamic(ctx) {
		t.Fatal("non-linux executable should not export dynamic symbols for funcinfo")
	}
	ctx.buildConf.Goos = "linux"
	ctx.buildConf.Target = "wasi"
	if needsLinuxExportDynamic(ctx) {
		t.Fatal("named targets should not force host linux dynamic symbol export")
	}
}

func TestIsFuncInfoEnabled(t *testing.T) {
	t.Setenv(llgoFuncInfo, "")
	if !IsFuncInfoEnabled() {
		t.Fatal("funcinfo should be enabled by default")
	}
	t.Setenv(llgoFuncInfo, "0")
	if IsFuncInfoEnabled() {
		t.Fatal("LLGO_FUNCINFO=0 should disable funcinfo")
	}
	t.Setenv(llgoFuncInfo, "1")
	if !IsFuncInfoEnabled() {
		t.Fatal("LLGO_FUNCINFO=1 should enable funcinfo")
	}
}

func TestLinkedModuleGlobalsSkipsDeclarations(t *testing.T) {
	prog := llssa.NewProgram(nil)
	lpkg := prog.NewPackage("example.com/p", "example.com/p")
	mod := lpkg.Module()
	i32 := mod.Context().Int32Type()

	defined := llvm.AddGlobal(mod, i32, "example.com/p.defined")
	defined.SetInitializer(llvm.ConstInt(i32, 1, false))
	llvm.AddGlobal(mod, i32, "example.com/p.declared")

	got := linkedModuleGlobals([]Package{{LPkg: lpkg}})
	if _, ok := got["example.com/p.defined"]; !ok {
		t.Fatalf("linkedModuleGlobals missing defined global: %#v", got)
	}
	if _, ok := got["example.com/p.declared"]; ok {
		t.Fatalf("linkedModuleGlobals should skip external declarations: %#v", got)
	}
}

func mockRun(args []string, cfg *Config) {
	defer mockable.DisableMock()
	mockable.EnableMock()

	var panicVal interface{}
	defer func() {
		if r := recover(); r != nil {
			// Ignore mocked os.Exit
			if s, ok := r.(string); ok && s == "exit" {
				return
			}
			panicVal = r
		}
		if panicVal != nil {
			panic(panicVal)
		}
	}()

	// Only set OutFile for modes that don't support multiple packages,
	// or when OutFile is not already set
	if cfg.OutFile == "" && (cfg.Mode == ModeBuild || cfg.Mode == ModeRun) {
		file, _ := os.CreateTemp("", "llgo-*")
		cfg.OutFile = file.Name()
		file.Close()
		defer os.Remove(cfg.OutFile)
	}

	if _, err := Do(args, cfg); err != nil {
		panic(err)
	}
}

func TestRun(t *testing.T) {
	mockRun([]string{"../../cl/_testgo/print"}, &Config{Mode: ModeRun})
}

func TestTest(t *testing.T) {
	// FIXME(zzy): with builtin package test in a llgo test ./... will cause duplicate symbol error
	mockRun([]string{"../../cl/_testgo/runtest"}, &Config{Mode: ModeTest})
}

func TestExtest(t *testing.T) {
	originalStdout := os.Stdout
	defer func() { os.Stdout = originalStdout }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	os.Stdout = w
	outputChan := make(chan string)
	go func() {
		var data bytes.Buffer
		io.Copy(&data, r)
		outputChan <- data.String()
	}()

	mockRun([]string{"../../cl/_testgo/runextest/..."}, &Config{Mode: ModeTest})

	w.Close()
	got := <-outputChan
	expected := "PASS\nPASS\nPASS\nPASS\n"
	if got != expected {
		t.Errorf("Expected output %q, but got %q", expected, got)
	}
}

func TestCmpTest(t *testing.T) {
	mockRun([]string{"../../cl/_testgo/runtest"}, &Config{Mode: ModeCmpTest})
}

func TestFilterTestPackages(t *testing.T) {
	pkg := func(id string) *packages.Package {
		return &packages.Package{ID: id}
	}

	t.Run("empty after filtering", func(t *testing.T) {
		initial := []*packages.Package{
			pkg("github.com/xgo-dev/llgo/chore/ardump"),
			pkg("github.com/xgo-dev/llgo/chore/ardump [github.com/xgo-dev/llgo/chore/ardump.test]"),
		}
		filtered, err := filterTestPackages(initial, "")
		if err != nil {
			t.Fatalf("filterTestPackages returned unexpected error: %v", err)
		}
		if len(filtered) != 0 {
			t.Fatalf("len(filtered) = %d, want 0", len(filtered))
		}
	})

	t.Run("retain test packages", func(t *testing.T) {
		initial := []*packages.Package{
			pkg("foo"),
			pkg("foo.test"),
		}
		filtered, err := filterTestPackages(initial, "")
		if err != nil {
			t.Fatalf("filterTestPackages returned unexpected error: %v", err)
		}
		if len(filtered) != 1 {
			t.Fatalf("len(filtered) = %d, want 1", len(filtered))
		}
		if filtered[0].ID != "foo.test" {
			t.Fatalf("filtered[0].ID = %q, want %q", filtered[0].ID, "foo.test")
		}
	})

	t.Run("multiple test packages with output file", func(t *testing.T) {
		initial := []*packages.Package{
			pkg("a.test"),
			pkg("b.test"),
		}
		_, err := filterTestPackages(initial, "/tmp/out")
		if err == nil {
			t.Fatal("expected error for -o with multiple test packages, got nil")
		}
		if !strings.Contains(err.Error(), "cannot use -o flag with multiple packages") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

const (
	rewriteMainPkg = "github.com/xgo-dev/llgo/cl/_testgo/rewrite"
	rewriteDepPkg  = rewriteMainPkg + "/dep"
	rewriteDirPath = "../../cl/_testgo/rewrite"
)

func TestLdFlagsRewriteVars(t *testing.T) {
	buildRewriteBinary(t, false, "build-main", "build-pkg")
	buildRewriteBinary(t, false, "rerun-main", "rerun-pkg")
}

func TestLdFlagsRewriteVarsMainAlias(t *testing.T) {
	buildRewriteBinary(t, true, "alias-main", "alias-pkg")
}

func TestLinkOptionsOmitDWARFPreservesPclntab(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("initial -s/-w backend coverage is limited to native Mach-O and ELF")
	}
	t.Setenv(llgoFuncInfo, "1")

	tests := []struct {
		name    string
		options LinkOptions
	}{
		{name: "w", options: LinkOptions{DWARF: DWARFOmit}},
		{name: "s", options: LinkOptions{OmitSymbolTable: true}},
		{name: "s_w_false", options: LinkOptions{OmitSymbolTable: true, DWARF: DWARFPreserve}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binPath := filepath.Join(t.TempDir(), "ldflagsstrip")
			cfg := &Config{
				Mode:        ModeBuild,
				OutFile:     binPath,
				LinkOptions: tt.options,
			}
			if _, err := Do([]string{"./testdata/ldflagsstrip"}, cfg); err != nil {
				t.Fatalf("ModeBuild with LinkOptions %+v failed: %v", tt.options, err)
			}
			if got, want := runBinary(t, binPath), "main.caller main.go true\n"; got != want {
				t.Fatalf("runtime symbolization with LinkOptions %+v:\nwant %q\ngot  %q", tt.options, want, got)
			}
			if runtime.GOOS == "darwin" {
				if out, err := exec.Command("codesign", "--verify", "--verbose=4", binPath).CombinedOutput(); err != nil {
					t.Fatalf("codesign verification with LinkOptions %+v: %v\n%s", tt.options, err, out)
				}
			}
		})
	}
}

func TestLinkOptionsControlELFDWARF(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("ELF DWARF integration test")
	}
	tests := []struct {
		name      string
		options   LinkOptions
		wantDWARF bool
	}{
		{name: "omit", options: LinkOptions{DWARF: DWARFOmit}},
		{name: "preserve", options: LinkOptions{DWARF: DWARFPreserve}, wantDWARF: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binPath := filepath.Join(t.TempDir(), "ldflagsstrip")
			cfg := &Config{Mode: ModeBuild, OutFile: binPath, LinkOptions: tt.options}
			if _, err := Do([]string{"./testdata/ldflagsstrip"}, cfg); err != nil {
				t.Fatalf("ModeBuild with LinkOptions %+v failed: %v", tt.options, err)
			}
			if got := elfHasDebugInfo(t, binPath); got != tt.wantDWARF {
				t.Fatalf("ELF DWARF with LinkOptions %+v = %v, want %v", tt.options, got, tt.wantDWARF)
			}
			_ = runBinary(t, binPath)
		})
	}
}

func TestLinkOptionsControlDarwinDebugSymbols(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Mach-O debug-symbol integration test")
	}
	tests := []struct {
		name      string
		options   LinkOptions
		wantSTABS bool
	}{
		{name: "omit", options: LinkOptions{DWARF: DWARFOmit}},
		{name: "preserve", options: LinkOptions{DWARF: DWARFPreserve}, wantSTABS: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binPath := filepath.Join(t.TempDir(), "ldflagsstrip")
			cfg := &Config{Mode: ModeBuild, OutFile: binPath, LinkOptions: tt.options}
			if _, err := Do([]string{"./testdata/ldflagsstrip"}, cfg); err != nil {
				t.Fatalf("ModeBuild with LinkOptions %+v failed: %v", tt.options, err)
			}
			if got := machoHasStabs(t, binPath); got != tt.wantSTABS {
				t.Fatalf("Mach-O STABS with LinkOptions %+v = %v, want %v", tt.options, got, tt.wantSTABS)
			}
			_ = runBinary(t, binPath)
			if out, err := exec.Command("codesign", "--verify", "--verbose=4", binPath).CombinedOutput(); err != nil {
				t.Fatalf("codesign verification with LinkOptions %+v: %v\n%s", tt.options, err, out)
			}
		})
	}
}

func machoHasStabs(t *testing.T, path string) bool {
	t.Helper()
	f, err := macho.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if f.Symtab == nil {
		return false
	}
	const nStab = 0xe0
	for _, symbol := range f.Symtab.Syms {
		if symbol.Type&nStab != 0 {
			return true
		}
	}
	return false
}

func TestDoRejectsInvalidLinkOptions(t *testing.T) {
	_, err := Do(nil, &Config{LinkOptions: LinkOptions{DWARF: DWARFMode(255)}})
	if err == nil || !strings.Contains(err.Error(), "invalid DWARF mode 255") {
		t.Fatalf("Do() error = %v, want invalid DWARF mode", err)
	}
}

func buildRewriteBinary(t *testing.T, useMainAlias bool, mainVal, depVal string) {
	t.Helper()
	binPath := filepath.Join(t.TempDir(), "rewrite")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	cfg := &Config{Mode: ModeBuild, OutFile: binPath}
	mainKey := rewriteMainPkg
	var mainPkgs []string
	if useMainAlias {
		mainKey = "main"
		mainPkgs = []string{rewriteMainPkg}
	}
	mainPlain := mainVal + "-plain"
	depPlain := depVal + "-plain"
	gorootVal := "goroot-" + mainVal
	versionVal := "version-" + mainVal
	addGlobalString(cfg, mainKey+".VarName="+mainVal, mainPkgs)
	addGlobalString(cfg, mainKey+".VarPlain="+mainPlain, mainPkgs)
	addGlobalString(cfg, rewriteDepPkg+".VarName="+depVal, nil)
	addGlobalString(cfg, rewriteDepPkg+".VarPlain="+depPlain, nil)
	addGlobalString(cfg, "runtime.defaultGOROOT="+gorootVal, nil)
	addGlobalString(cfg, "runtime.buildVersion="+versionVal, nil)

	if _, err := Do([]string{rewriteDirPath}, cfg); err != nil {
		t.Fatalf("ModeBuild failed: %v", err)
	}
	got := runBinary(t, binPath)
	want := fmt.Sprintf(
		"main.VarName: %s\nmain.VarPlain: %s\ndep.VarName: %s\ndep.VarPlain: %s\nruntime.GOROOT(): %s\nruntime.Version(): %s\n",
		mainVal, mainPlain, depVal, depPlain, gorootVal, versionVal,
	)
	if got != want {
		t.Fatalf("unexpected binary output:\nwant %q\ngot  %q", want, got)
	}
}

func runBinary(t *testing.T, path string, args ...string) string {
	t.Helper()
	cmd := exec.Command(path, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run %s: %v\n%s", path, err, output)
	}
	return string(output)
}

func TestRunPrintfWithStdioNobuf(t *testing.T) {
	t.Setenv(llgoStdioNobuf, "1")
	mockRun([]string{"../../cl/_testdata/printf"}, &Config{Mode: ModeRun})
}

func TestTestOutputFileLogic(t *testing.T) {
	// Test output file path determination logic for test mode
	tests := []struct {
		name        string
		pkgName     string
		conf        *Config
		multiPkg    bool
		wantBase    string
		wantDir     string
		description string
	}{
		{
			name:        "compile only without -o",
			pkgName:     "mypackage.test",
			conf:        &Config{Mode: ModeTest, CompileOnly: true},
			multiPkg:    false,
			wantBase:    "mypackage.test",
			wantDir:     ".",
			description: "-c without -o: write pkg.test in current directory",
		},
		{
			name:        "with -o absolute file path",
			pkgName:     "mypackage",
			conf:        &Config{Mode: ModeTest, OutFile: "/tmp/mytest.test", AppExt: ".test"},
			multiPkg:    false,
			wantBase:    "mytest",
			wantDir:     "/tmp",
			description: "-o with absolute file path: use specified file",
		},
		{
			name:        "with -o relative file path",
			pkgName:     "mypackage",
			conf:        &Config{Mode: ModeTest, OutFile: "my.test", AppExt: ".test"},
			multiPkg:    false,
			wantBase:    "my",
			wantDir:     ".",
			description: "-o with relative file path: use specified file in current dir",
		},
		{
			name:        "with -o directory",
			pkgName:     "mypackage.test",
			conf:        &Config{Mode: ModeTest, OutFile: "/tmp/build/", AppExt: ".test"},
			multiPkg:    false,
			wantBase:    "mypackage.test",
			wantDir:     "/tmp/build/",
			description: "-o with directory: write pkg.test in that directory",
		},
		{
			name:        "default test mode",
			pkgName:     "mypackage",
			conf:        &Config{Mode: ModeTest, AppExt: ".test"},
			multiPkg:    false,
			wantBase:    "mypackage",
			wantDir:     "",
			description: "default test mode: use temp file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseName, dir := determineBaseNameAndDir(tt.pkgName, tt.conf, tt.multiPkg)
			if baseName != tt.wantBase {
				t.Errorf("%s: got baseName=%q, want %q", tt.description, baseName, tt.wantBase)
			}
			if dir != tt.wantDir {
				t.Errorf("%s: got dir=%q, want %q", tt.description, dir, tt.wantDir)
			}
		})
	}
}

func TestTestMultiplePackagesWithOutputFile(t *testing.T) {
	// Test that -o flag errors with multiple test packages
	cfg := &Config{
		Mode:    ModeTest,
		OutFile: "/tmp/output",
	}

	// Create a scenario that would have multiple test packages
	// This should error during Do() validation
	args := []string{"../../cl/_testgo/runextest/..."}
	_, err := Do(args, cfg)
	if err == nil {
		t.Fatal("Expected error when using -o flag with multiple packages, got nil")
	}
	if !strings.Contains(err.Error(), "cannot use -o flag with multiple packages") {
		t.Errorf("Expected error about -o with multiple packages, got: %v", err)
	}
}

func TestCmpTestNonexistentPatternReturnsError(t *testing.T) {
	cfg := &Config{Mode: ModeCmpTest}
	_, err := Do([]string{"./this/path/does/not/exist/..."}, cfg)
	if err == nil {
		t.Fatal("expected error for nonexistent cmptest pattern")
	}
	if !strings.Contains(err.Error(), "cannot build SSA for packages") && !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParsePkgSyntaxCollectsRuntimeLinknames(t *testing.T) {
	prog := llssa.NewProgram(nil)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "runtime.go", `package runtime
import _ "unsafe"
//go:linkname Sigsetjmp C.sigsetjmp
func Sigsetjmp()
`, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	pkg := types.NewPackage(llssa.PkgRuntime, "runtime")
	if err := cl.ParsePkgSyntax(prog, fset, pkg, []*ast.File{file}); err != nil {
		t.Fatal(err)
	}
	if got, ok := prog.Linkname(llssa.PkgRuntime + ".Sigsetjmp"); !ok || got != "C.sigsetjmp" {
		t.Fatalf("pre-collected runtime linkname = (%q,%v), want (%q,%v)", got, ok, "C.sigsetjmp", true)
	}
}

func TestPrepareLocalVariables(t *testing.T) {
	newLocalPackage := func(path string, withSyntax bool) (*packages.Package, *ast.File) {
		pkg := types.NewPackage(path, "local")
		value := types.NewVar(token.NoPos, pkg, "value", types.Typ[types.Int])
		pkg.Scope().Insert(value)
		info := &types.Info{
			Defs:      make(map[*ast.Ident]types.Object),
			Uses:      make(map[*ast.Ident]types.Object),
			InitOrder: []*types.Initializer{{Lhs: []*types.Var{value}, Rhs: ast.NewIdent("rhs")}},
		}
		loaded := &packages.Package{Types: pkg, TypesInfo: info}
		var file *ast.File
		if withSyntax {
			file = &ast.File{Name: ast.NewIdent("local")}
			loaded.Syntax = []*ast.File{file}
		}
		return loaded, file
	}

	t.Run("accepts no package groups", func(t *testing.T) {
		if err := prepareLocalVariables(llssa.NewProgram(nil)); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("filters and deduplicates packages", func(t *testing.T) {
		prog := llssa.NewProgram(nil)
		loaded, file := newLocalPackage("example.com/local", true)
		prog.SetLocalityInfo("example.com/local.value", llssa.LocalityInfo{Locality: llssa.ThreadLocal, HasInitializer: true})
		duplicate := *loaded

		err := prepareLocalVariables(prog,
			[]*packages.Package{{}, {Types: types.NewPackage("example.com/bad", "bad"), IllTyped: true}, loaded},
			[]*packages.Package{&duplicate},
		)
		if err != nil {
			t.Fatal(err)
		}
		if got := len(file.Decls); got != 1 {
			t.Fatalf("generated initializer declarations = %d, want 1", got)
		}
	})

	t.Run("returns dependency error", func(t *testing.T) {
		prog := llssa.NewProgram(nil)
		dependency, _ := newLocalPackage("example.com/dependency", false)
		prog.SetLocalityInfo("example.com/dependency.value", llssa.LocalityInfo{Locality: llssa.GoroutineLocal, HasInitializer: true})
		root := &packages.Package{
			Types:   types.NewPackage("example.com/root", "root"),
			Imports: map[string]*packages.Package{"example.com/dependency": dependency},
		}

		err := prepareLocalVariables(prog, []*packages.Package{root})
		if err == nil || !strings.Contains(err.Error(), "without syntax files") {
			t.Fatalf("prepareLocalVariables error = %v", err)
		}
	})

	t.Run("skips inactive alternate roots", func(t *testing.T) {
		prog := llssa.NewProgram(nil)
		active := &packages.Package{Types: types.NewPackage("example.com/active", "active")}
		inactive := &packages.Package{Types: types.NewPackage("example.com/inactive", "inactive")}
		err := prepareLocalVariables(prog,
			[]*packages.Package{active},
			[]*packages.Package{{}, inactive},
		)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestPrepareLocalVariablesKeepsAltDeclarationOwners(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "runtime.go", `package runtime

//llgo:gls
var goroutineState *uint32

//llgo:tls
var threadState uintptr
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
		Instances:  make(map[*ast.Ident]types.Instance),
	}
	alt, err := (&types.Config{}).Check(altPkgPathPrefix+"runtime", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}

	prog := llssa.NewProgram(nil)
	if err := cl.ParsePkgSyntax(prog, fset, alt, []*ast.File{file}); err != nil {
		t.Fatal(err)
	}
	std := types.NewPackage("runtime", "runtime")
	err = prepareLocalVariables(prog,
		[]*packages.Package{{Types: std, TypesInfo: &types.Info{}}},
		[]*packages.Package{{Types: alt, TypesInfo: info, Syntax: []*ast.File{file}, Fset: fset}},
	)
	if err != nil {
		t.Fatalf("prepareLocalVariables confused standard and alternate runtime packages: %v", err)
	}
	for name, want := range map[string]llssa.VariableLocality{
		"runtime.goroutineState": {
			Info:         llssa.LocalityInfo{Locality: llssa.GoroutineLocal},
			LocalStorage: llssa.LocalStoragePackage,
		},
		"runtime.threadState": {
			Info:         llssa.LocalityInfo{Locality: llssa.ThreadLocal},
			LocalStorage: llssa.LocalStorageNativeTLS,
		},
	} {
		got, ok := prog.VariableLocality(name)
		if !ok || got.Locality != want.Locality || got.LocalStorage != want.LocalStorage {
			t.Fatalf("%s locality = %+v, %v", name, got, ok)
		}
	}
	if !prog.NeedsLocalContext() {
		t.Fatal("active alternate runtime package did not require a local context")
	}
}

func TestLTOEnabledDefault(t *testing.T) {
	host := &Config{Target: ""}
	if host.ltoEnabled() {
		t.Fatal("expected LTO disabled by default for non-target builds")
	}

	target := &Config{Target: "rp2040"}
	if target.ltoEnabled() {
		t.Fatal("expected LTO disabled by default for target builds")
	}
}

func TestLTOEnabledExplicitOverride(t *testing.T) {
	hostOn := &Config{Target: "", LTO: lto.Thin}
	if !hostOn.ltoEnabled() {
		t.Fatal("expected explicit LTO=thin to enable LTO for non-target build")
	}

	hostFull := &Config{Target: "", LTO: lto.Full}
	if !hostFull.ltoEnabled() {
		t.Fatal("expected explicit LTO=full to enable LTO for non-target build")
	}

	targetOff := &Config{Target: "rp2040", LTO: lto.Off}
	if targetOff.ltoEnabled() {
		t.Fatal("expected LTO=off to disable LTO for target build")
	}
}

func TestArchiverPrefersLLVMArForLTO(t *testing.T) {
	td := t.TempDir()
	llvmAr := filepath.Join(td, "llvm-ar")
	if err := os.WriteFile(llvmAr, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", td)
	t.Setenv("LLGO_AR", "")

	if got := (&context{buildConf: &Config{LTO: lto.Off}}).archiver(); got != "ar" {
		t.Fatalf("archiver without lto = %q, want ar", got)
	}
	if got := (&context{buildConf: &Config{LTO: lto.Full}}).archiver(); got != llvmAr {
		t.Fatalf("archiver with full lto = %q, want %q", got, llvmAr)
	}
}

func TestArchiverAllowsLLGOAROverrideForLTO(t *testing.T) {
	t.Setenv("LLGO_AR", "custom-ar")

	if got := (&context{buildConf: &Config{LTO: lto.Full}}).archiver(); got != "custom-ar" {
		t.Fatalf("archiver with LLGO_AR = %q, want custom-ar", got)
	}
}

func TestCSharedExportArgs(t *testing.T) {
	if got := cSharedExportArgs(nil, nil); got != nil {
		t.Fatalf("nil cSharedExportArgs = %v, want nil", got)
	}
	prog := llssa.NewProgram(nil)
	lpkg := prog.NewPackage("example.com/p", "example.com/p")
	lpkg.SetExport("example.com/p.Z", "Zed")
	lpkg.SetExport("example.com/p.A", "Add")
	pkgs := []*aPackage{{LPkg: lpkg}}

	ctx := &context{buildConf: &Config{BuildMode: BuildModeCShared, Goos: "linux"}}
	if got, want := strings.Join(cSharedExportArgs(ctx, pkgs), " "), "-Wl,--undefined=Add -Wl,--undefined=Zed"; got != want {
		t.Fatalf("linux cSharedExportArgs = %q, want %q", got, want)
	}
	ctx.buildConf.Goos = "darwin"
	if got, want := strings.Join(cSharedExportArgs(ctx, pkgs), " "), "-Wl,-u,_Add -Wl,-u,_Zed"; got != want {
		t.Fatalf("darwin cSharedExportArgs = %q, want %q", got, want)
	}
	ctx.buildConf.BuildMode = BuildModeExe
	if got := cSharedExportArgs(ctx, pkgs); got != nil {
		t.Fatalf("executable cSharedExportArgs = %v, want nil", got)
	}
}

func TestCSharedExportArgsKeepsTestMain(t *testing.T) {
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	lpkg := prog.NewPackage("example.com/p.test", "example.com/p.test")
	pkgs := []*aPackage{{
		Package: &packages.Package{
			Name:    "main",
			PkgPath: "example.com/p.test",
		},
		LPkg: lpkg,
	}}
	ctx := &context{
		mode:      ModeTest,
		buildConf: &Config{BuildMode: BuildModeCShared, Goos: "linux"},
	}
	if got, want := strings.Join(cSharedExportArgs(ctx, pkgs), " "), "-Wl,--undefined=main.init -Wl,--undefined=main.main"; got != want {
		t.Fatalf("test main cSharedExportArgs = %q, want %q", got, want)
	}
}

func TestApplyBuildModeCompileFlags(t *testing.T) {
	tests := []struct {
		name string
		mode BuildMode
		in   []string
		want string
	}{
		{name: "shared adds PIC", mode: BuildModeCShared, want: "-fPIC"},
		{name: "shared preserves flags", mode: BuildModeCShared, in: []string{"-O2"}, want: "-O2 -fPIC"},
		{name: "shared does not duplicate PIC", mode: BuildModeCShared, in: []string{"-fPIC"}, want: "-fPIC"},
		{name: "archive remains unchanged", mode: BuildModeCArchive, in: []string{"-O2"}, want: "-O2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			export := crosscompile.Export{CCFLAGS: slices.Clone(tt.in)}
			applyBuildModeCompileFlags(tt.mode, &export)
			if got := strings.Join(export.CCFLAGS, " "); got != tt.want {
				t.Fatalf("CCFLAGS = %q, want %q", got, tt.want)
			}
		})
	}

	applyBuildModeCompileFlags(BuildModeCShared, nil)
}

func TestCHeaderPackagesExcludesStandardRuntime(t *testing.T) {
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	userLPkg := prog.NewPackage("example.com/p", "example.com/p")
	userLPkg.SetExport("example.com/p.Export", "Export")
	runtimeLPkg := prog.NewPackage("runtime", "runtime")
	llgoRuntimeLPkg := prog.NewPackage("github.com/xgo-dev/llgo/runtime/internal/lib/runtime", "github.com/xgo-dev/llgo/runtime/internal/lib/runtime")
	dependencyLPkg := prog.NewPackage("example.com/dep", "example.com/dep")
	pkgs := []*aPackage{
		{Package: &packages.Package{PkgPath: "example.com/p"}, LPkg: userLPkg},
		{Package: &packages.Package{PkgPath: "runtime"}, LPkg: runtimeLPkg},
		{Package: &packages.Package{PkgPath: "github.com/xgo-dev/llgo/runtime/internal/lib/runtime"}, LPkg: llgoRuntimeLPkg},
		{Package: &packages.Package{PkgPath: "example.com/dep"}, LPkg: dependencyLPkg},
		nil,
	}
	got := cHeaderPackages(pkgs)
	if len(got) != 1 || got[0] != userLPkg {
		t.Fatalf("cHeaderPackages = %v, want only user package", got)
	}

	if hasLocalCExports(nil) {
		t.Fatal("hasLocalCExports(nil) = true, want false")
	}
	unqualifiedLPkg := prog.NewPackage("example.com/unqualified", "example.com/unqualified")
	unqualifiedLPkg.SetExport("Export", "Export")
	if !hasLocalCExports(unqualifiedLPkg) {
		t.Fatal("hasLocalCExports(unqualified) = false, want true")
	}
	foreignOnlyLPkg := prog.NewPackage("example.com/foreign", "example.com/foreign")
	foreignOnlyLPkg.SetExport("runtime.Export", "Export")
	if hasLocalCExports(foreignOnlyLPkg) {
		t.Fatal("hasLocalCExports(foreign only) = true, want false")
	}
}

func TestArchiveMergerSelection(t *testing.T) {
	t.Run("override", func(t *testing.T) {
		t.Setenv("LLGO_AR", "custom-llvm-ar")
		got, err := (&context{}).archiveMerger()
		if err != nil || got != "custom-llvm-ar" {
			t.Fatalf("archiveMerger() = %q, %v, want custom-llvm-ar", got, err)
		}
	})

	t.Run("next to compiler", func(t *testing.T) {
		t.Setenv("LLGO_AR", "")
		t.Setenv("PATH", "")
		td := t.TempDir()
		llvmAr := filepath.Join(td, "llvm-ar")
		if err := os.WriteFile(llvmAr, nil, 0o755); err != nil {
			t.Fatal(err)
		}
		ctx := &context{}
		ctx.crossCompile.CC = filepath.Join(td, "clang")
		got, err := ctx.archiveMerger()
		if err != nil || got != llvmAr {
			t.Fatalf("archiveMerger() = %q, %v, want %q", got, err, llvmAr)
		}
	})

	t.Run("path", func(t *testing.T) {
		t.Setenv("LLGO_AR", "")
		td := t.TempDir()
		llvmAr := filepath.Join(td, "llvm-ar")
		if err := os.WriteFile(llvmAr, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", td)
		got, err := (&context{}).archiveMerger()
		if err != nil || got != llvmAr {
			t.Fatalf("archiveMerger() = %q, %v, want %q", got, err, llvmAr)
		}
	})

	t.Run("missing", func(t *testing.T) {
		t.Setenv("LLGO_AR", "")
		t.Setenv("PATH", "")
		if got, err := (&context{}).archiveMerger(); err == nil || got != "" {
			t.Fatalf("archiveMerger() = %q, %v, want missing-tool error", got, err)
		}
	})
}

func TestCreateMergedArchiveFileFlattensInputs(t *testing.T) {
	llvmAr, err := exec.LookPath("llvm-ar")
	if err != nil {
		t.Skip("llvm-ar is not installed")
	}
	t.Setenv("LLGO_AR", llvmAr)

	td := filepath.Join(t.TempDir(), "archive with spaces")
	if err := os.MkdirAll(td, 0o755); err != nil {
		t.Fatal(err)
	}
	direct := filepath.Join(td, "direct.o")
	nestedOne := filepath.Join(td, "nested-one.o")
	nestedTwo := filepath.Join(td, "nested-two.o")
	for path, content := range map[string]string{
		direct:    "direct",
		nestedOne: "nested one",
		nestedTwo: "nested two",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	nested := filepath.Join(td, "nested.a")
	if output, err := exec.Command(llvmAr, "rcs", nested, nestedOne, nestedTwo).CombinedOutput(); err != nil {
		t.Fatalf("create nested archive: %v\n%s", err, output)
	}

	out := filepath.Join(td, "combined.a")
	ctx := &context{buildConf: &Config{}}
	if err := ctx.createMergedArchiveFile(out, []string{direct, nested}, true); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(llvmAr, "t", out).CombinedOutput()
	if err != nil {
		t.Fatalf("list merged archive: %v\n%s", err, output)
	}
	members := strings.Fields(string(output))
	slices.Sort(members)
	if got, want := strings.Join(members, " "), "direct.o nested-one.o nested-two.o"; got != want {
		t.Fatalf("merged archive members = %q, want %q", got, want)
	}
}

func TestCreateMergedArchiveFileErrors(t *testing.T) {
	ctx := &context{buildConf: &Config{}}
	if err := ctx.createMergedArchiveFile(filepath.Join(t.TempDir(), "empty.a"), nil); err == nil {
		t.Fatal("createMergedArchiveFile with no inputs succeeded")
	}

	td := t.TempDir()
	failingAr := filepath.Join(td, "llvm-ar")
	if err := os.WriteFile(failingAr, []byte("#!/bin/sh\necho merge failed >&2\nexit 7\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_AR", failingAr)
	input := filepath.Join(td, "input.o")
	if err := os.WriteFile(input, []byte("object"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ctx.createMergedArchiveFile(filepath.Join(td, "failed.a"), []string{input}); err == nil || !strings.Contains(err.Error(), "merge failed") {
		t.Fatalf("createMergedArchiveFile error = %v, want archiver output", err)
	}
}

func TestDevLTOGlobalDCEDefaultsToFullLTO(t *testing.T) {
	tests := []struct {
		name string
		conf *Config
		want bool
	}{
		{name: "lto off", conf: &Config{LTO: lto.Off}, want: false},
		{name: "thin lto", conf: &Config{LTO: lto.Thin}, want: false},
		{name: "full lto", conf: &Config{LTO: lto.Full}, want: buildenv.Dev},
		{name: "full lto disabled", conf: &Config{LTO: lto.Full, DisableGoGlobalDCE: true}, want: false},
		{name: "disabled without full lto", conf: &Config{LTO: lto.Off, DisableGoGlobalDCE: true}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.conf.goGlobalDCEEnabled(); got != tt.want {
				t.Fatalf("goGlobalDCEEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeadcodeDropEnabled(t *testing.T) {
	tests := []struct {
		name string
		conf *Config
		want bool
	}{
		{name: "not requested", conf: &Config{LTO: lto.Off}, want: false},
		{name: "requested", conf: &Config{DeadcodeDrop: true, LTO: lto.Off}, want: buildenv.Dev},
		{name: "disabled by go global dce", conf: &Config{DeadcodeDrop: true, LTO: lto.Full}, want: false},
		{name: "enabled when go global dce disabled", conf: &Config{DeadcodeDrop: true, LTO: lto.Full, DisableGoGlobalDCE: true}, want: buildenv.Dev},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.conf.deadcodeDropEnabled(); got != tt.want {
				t.Fatalf("deadcodeDropEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestThinLTODeadcodeEnabled(t *testing.T) {
	tests := []struct {
		name string
		conf *Config
		want bool
	}{
		{name: "not requested", conf: &Config{LTO: lto.Thin}, want: false},
		{name: "thin lto", conf: &Config{DeadcodeDrop: true, LTO: lto.Thin}, want: buildenv.Dev},
		{name: "lto off", conf: &Config{DeadcodeDrop: true, LTO: lto.Off}, want: false},
		{name: "full lto", conf: &Config{DeadcodeDrop: true, LTO: lto.Full, DisableGoGlobalDCE: true}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.conf.thinLTODeadcodeEnabled(); got != tt.want {
				t.Fatalf("thinLTODeadcodeEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestThinLTOFeedbackEnabled(t *testing.T) {
	conf := &Config{Goos: "linux", DeadcodeDrop: true, LTO: lto.Thin, BuildMode: BuildModeExe}
	if got := conf.thinLTOFeedbackEnabled(); got {
		t.Fatal("feedback enabled without LLGO_THINLTO_FEEDBACK=1")
	}
	t.Setenv("LLGO_THINLTO_FEEDBACK", "1")
	want := buildenv.Dev
	if got := conf.thinLTOFeedbackEnabled(); got != want {
		t.Fatalf("thinLTOFeedbackEnabled() = %v, want %v", got, want)
	}
	conf.LTO = lto.Full
	if conf.thinLTOFeedbackEnabled() {
		t.Fatal("feedback enabled for full LTO")
	}
}

func TestThinLTODeadcodeLinkerArgs(t *testing.T) {
	tests := []struct {
		name string
		conf *Config
		want []string
	}{
		{name: "not requested", conf: &Config{LTO: lto.Thin}},
		{name: "lto off", conf: &Config{DeadcodeDrop: true, LTO: lto.Off}},
		{name: "full lto", conf: &Config{DeadcodeDrop: true, LTO: lto.Full, DisableGoGlobalDCE: true}},
		{name: "thin lto deadcode", conf: &Config{DeadcodeDrop: true, LTO: lto.Thin}, want: []string{thinLTODeadcodeImportLimitFlag}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := tt.want
			if !buildenv.Dev {
				want = nil
			}
			if got := thinLTODeadcodeLinkerArgs(tt.conf); !reflect.DeepEqual(got, want) {
				t.Fatalf("thinLTODeadcodeLinkerArgs() = %v, want %v", got, want)
			}
		})
	}
}

func TestPackageMetaEnabled(t *testing.T) {
	tests := []struct {
		name string
		conf *Config
		want bool
	}{
		{name: "disabled", conf: &Config{}, want: false},
		{name: "explicit collection", conf: &Config{CollectPackageMeta: true}, want: true},
		{name: "deadcode drop", conf: &Config{DeadcodeDrop: true}, want: buildenv.Dev},
		{name: "collection with global dce", conf: &Config{CollectPackageMeta: true, LTO: lto.Full}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.conf.packageMetaEnabled(); got != tt.want {
				t.Fatalf("packageMetaEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAllowMissingFunctionBodies(t *testing.T) {
	pkg := &packages.Package{
		Errors: []packages.Error{
			{Msg: "# command-line-arguments"},
			{Msg: "missing function body"},
		},
		IllTyped: true,
	}
	allowMissingFunctionBodies([]*packages.Package{pkg})
	if pkg.IllTyped || len(pkg.Errors) != 0 || len(pkg.TypeErrors) != 0 {
		t.Fatalf("package remains ill-typed: %+v", pkg.Errors)
	}

	pkg = &packages.Package{
		Errors: []packages.Error{
			{Msg: "missing function body"},
			{Msg: "undefined: missing"},
		},
		IllTyped: true,
	}
	allowMissingFunctionBodies([]*packages.Package{pkg})
	if !pkg.IllTyped || len(pkg.Errors) != 2 {
		t.Fatalf("mixed errors were incorrectly suppressed: %+v", pkg.Errors)
	}

	unchanged := &packages.Package{Errors: []packages.Error{{Msg: "# command-line-arguments"}}, IllTyped: true}
	allowMissingFunctionBodies([]*packages.Package{unchanged})
	if !unchanged.IllTyped || len(unchanged.Errors) != 1 {
		t.Fatalf("package without a missing-body diagnostic was changed: %+v", unchanged)
	}
}

func TestDoAllowsMissingFunctionBodies(t *testing.T) {
	file := filepath.Join(t.TempDir(), "nobody.go")
	if err := os.WriteFile(file, []byte(`package nobody

func External()

func F() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	conf := NewDefaultConf(ModeGen)
	conf.AllowNoBody = true
	pkgs, err := Do([]string{file}, conf)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].LPkg == nil {
		t.Fatalf("Do returned packages = %+v, want one compiled package", pkgs)
	}
	pkgs[0].LPkg.Prog.Dispose()
}

func TestDoOptimizesUnreachableBodylessCalls(t *testing.T) {
	conf := NewDefaultConf(ModeGen)
	conf.AllowNoBody = true
	pkgs, err := Do([]string{"./testdata/unreachablebodyless/main.go"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].LPkg == nil {
		t.Fatalf("Do returned packages = %+v, want one compiled package", pkgs)
	}
	defer pkgs[0].LPkg.Prog.Dispose()
	mod := pkgs[0].LPkg.Module()
	mod.SetDataLayout(pkgs[0].LPkg.Prog.DataLayout())
	mod.SetTarget(pkgs[0].LPkg.Prog.Target().Spec().Triple)
	pbo := llvm.NewPassBuilderOptions()
	defer pbo.Dispose()
	if err := mod.RunPasses("default<O2>", pkgs[0].LPkg.Prog.TargetMachine(), pbo); err != nil {
		t.Fatalf("optimize generated module: %v", err)
	}
	if ir := mod.String(); strings.Contains(ir, "call void @main.fail") {
		t.Fatalf("unreachable bodyless call remains after optimization:\n%s", ir)
	}
}

func TestDoReportsLocalityDirectiveError(t *testing.T) {
	file := filepath.Join(t.TempDir(), "invalid_locality.go")
	if err := os.WriteFile(file, []byte(`package invalidlocality

//llgo:tls
func Invalid() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	conf := NewDefaultConf(ModeGen)
	if _, err := Do([]string{file}, conf); err == nil || !strings.Contains(err.Error(), "applies only to package-level var declarations") {
		t.Fatalf("Do error = %v, want locality directive diagnostic", err)
	}
}

func TestDoRejectsLocalityLinkname(t *testing.T) {
	file := filepath.Join(t.TempDir(), "invalid_locality_alias.go")
	if err := os.WriteFile(file, []byte(`package invalidlocalityalias

import _ "unsafe"

//llgo:tls
var target int

//go:linkname alias example.com/target.value
//llgo:tls
var alias = 1
`), 0o644); err != nil {
		t.Fatal(err)
	}
	conf := NewDefaultConf(ModeGen)
	if _, err := Do([]string{file}, conf); err == nil || !strings.Contains(err.Error(), "cannot apply to a //go:linkname variable") {
		t.Fatalf("Do error = %v, want locality linkname diagnostic", err)
	}
}

func TestDoReportsAltPackageLocalityDirectiveError(t *testing.T) {
	root := t.TempDir()
	runtimeDir := filepath.Join(root, "runtime")
	runtimePkgDir := filepath.Join(runtimeDir, "internal", "runtime")
	if err := os.MkdirAll(runtimePkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runtimeDir, "go.mod"), []byte("module github.com/xgo-dev/llgo/runtime\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runtimePkgDir, "runtime.go"), []byte(`package runtime

//llgo:gls
func Invalid() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "main.go")
	if err := os.WriteFile(file, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	conf := NewDefaultConf(ModeGen)
	if _, err := Do([]string{file}, conf); err == nil || !strings.Contains(err.Error(), "applies only to package-level var declarations") {
		t.Fatalf("Do error = %v, want alternate-package locality directive diagnostic", err)
	}
}

func TestFormatPackageError(t *testing.T) {
	tests := []struct {
		name     string
		err      packages.Error
		noColumn bool
		want     string
	}{
		{name: "keep columns", err: packages.Error{Pos: "case.go:2:3", Msg: "bad"}, want: "case.go:2:3: bad"},
		{name: "remove column", err: packages.Error{Pos: "case.go:2:3", Msg: "bad"}, noColumn: true, want: "case.go:2: bad"},
		{name: "driver diagnostic", err: packages.Error{Pos: "-", Msg: "# command-line-arguments\ndriver detail\ncase.go:2:3: bad"}, noColumn: true, want: "-: # command-line-arguments\ndriver detail\ncase.go:2: bad"},
		{name: "empty position", err: packages.Error{Msg: "bad"}, noColumn: true, want: "-: bad"},
		{name: "dash position", err: packages.Error{Pos: "-", Msg: "bad"}, noColumn: true, want: "-: bad"},
		{name: "missing separators", err: packages.Error{Pos: "case.go", Msg: "bad"}, noColumn: true, want: "case.go: bad"},
		{name: "invalid column", err: packages.Error{Pos: "case.go:2:x", Msg: "bad"}, noColumn: true, want: "case.go:2:x: bad"},
		{name: "missing line separator", err: packages.Error{Pos: "2:3", Msg: "bad"}, noColumn: true, want: "2:3: bad"},
		{name: "invalid line", err: packages.Error{Pos: "case.go:x:3", Msg: "bad"}, noColumn: true, want: "case.go:x:3: bad"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatPackageError(tt.err, tt.noColumn); got != tt.want {
				t.Fatalf("formatPackageError() = %q, want %q", got, tt.want)
			}
		})
	}
}
