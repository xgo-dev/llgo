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
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/goplus/llgo/cl"
	"github.com/goplus/llgo/internal/buildenv"
	"github.com/goplus/llgo/internal/crosscompile"
	"github.com/goplus/llgo/internal/env"
	"github.com/goplus/llgo/internal/lto"
	"github.com/goplus/llgo/internal/meta"
	"github.com/goplus/llgo/internal/mockable"
	"github.com/goplus/llgo/internal/packages"
	llssa "github.com/goplus/llgo/ssa"
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
		{name: "configured wasm target", goarch: "wasm", target: "wasip1", want: base + ",nogc"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := defaultBuildTags(test.goarch, test.target); got != test.want {
				t.Fatalf("defaultBuildTags(%q, %q) = %q, want %q", test.goarch, test.target, got, test.want)
			}
		})
	}
}

func TestEffectiveWasmTypeSizes(t *testing.T) {
	goSizes := types.SizesFor("gc", "wasm")
	for _, test := range []struct {
		name   string
		goos   string
		target string
		want   int64
	}{
		{name: "Go js wasm", goos: "js", want: 8},
		{name: "configured wasm", goos: "js", target: "wasm", want: 4},
		{name: "WASI compatibility", goos: "wasip1", want: 4},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := effectiveTypeSizes(goSizes, test.goos, "wasm", test.target)
			if size := got.Sizeof(types.Typ[types.Uintptr]); size != test.want {
				t.Fatalf("uintptr size = %d, want %d", size, test.want)
			}
		})
	}
	if got := effectiveTypeSizes(goSizes, "linux", "amd64", ""); got != goSizes {
		t.Fatal("native type sizes changed")
	}
}

func TestConfigureWasmGC(t *testing.T) {
	t.Setenv("LLGO_WASI_THREADS", "0")
	tests := []struct {
		name string
		conf Config
		want bool
		err  bool
	}{
		{name: "wasm32", conf: Config{Goos: "js", Goarch: "wasm", Tags: "llgo_wasm_gc"}, want: true},
		{name: "comma separated tags", conf: Config{Goos: "js", Goarch: "wasm", Tags: "other,llgo_wasm_gc"}, want: true},
		{name: "default wasm", conf: Config{Goos: "js", Goarch: "wasm"}},
		{name: "WASI", conf: Config{Goos: "wasip1", Goarch: "wasm", Tags: "llgo_wasm_gc"}},
		{name: "unsupported host", conf: Config{Goos: "linux", Goarch: "wasm", Tags: "llgo_wasm_gc"}, err: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			export := crosscompile.Export{}
			err := configureWasmGC(&test.conf, &export)
			if (err != nil) != test.err {
				t.Fatalf("configureWasmGC error = %v, want error %v", err, test.err)
			}
			if got := slices.Contains(export.LDFLAGS, "-sMALLOC=none"); got != test.want {
				t.Fatalf("MALLOC=none present = %v, want %v", got, test.want)
			}
		})
	}
}

func TestConfigureWasmGCRejectsWASIThreads(t *testing.T) {
	t.Setenv("LLGO_WASI_THREADS", "1")
	conf := Config{Goos: "wasip1", Goarch: "wasm", Tags: "llgo_wasm_gc"}
	if err := configureWasmGC(&conf, &crosscompile.Export{}); err == nil {
		t.Fatal("expected llgo_wasm_gc with WASI threads to fail")
	}
}

func TestWasmRuntimeAvoidsNativeHostDependencies(t *testing.T) {
	runtimeDir := filepath.Join(env.LLGoRuntimeDir(), "internal", "lib", "runtime")
	for _, goos := range []string{"js", "wasip1"} {
		t.Run(goos, func(t *testing.T) {
			ctx := gobuild.Default
			ctx.GOOS = goos
			ctx.GOARCH = "wasm"
			ctx.BuildTags = []string{"llgo", "nogc", "llgo_wasm_gc"}
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
					case "github.com/goplus/llgo/runtime/internal/clite/libuv",
						"github.com/goplus/llgo/runtime/internal/clite/bdwgc":
						t.Fatalf("wasm selected %s, which imports native host dependency %s", name, path)
					}
				}
			}

			for _, name := range []string{
				"mfinal_wasm.go",
				"runtime_baremetal.go",
				"runtime_gc_nonmoving.go",
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
			pkg("github.com/goplus/llgo/chore/ardump"),
			pkg("github.com/goplus/llgo/chore/ardump [github.com/goplus/llgo/chore/ardump.test]"),
		}
		filtered, err := filterTestPackages(initial, "", false)
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
		filtered, err := filterTestPackages(initial, "", false)
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
		_, err := filterTestPackages(initial, "/tmp/out", false)
		if err == nil {
			t.Fatal("expected error for -o with multiple test packages, got nil")
		}
		if !strings.Contains(err.Error(), "cannot use -o flag with multiple packages") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

const (
	rewriteMainPkg = "github.com/goplus/llgo/cl/_testgo/rewrite"
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
	if got, want := strings.Join(cSharedExportArgs(ctx, pkgs), " "), "-Wl,--undefined=example.com/p.test.init -Wl,--undefined=example.com/p.test.main"; got != want {
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

func TestWASIThreadsAreOptIn(t *testing.T) {
	t.Setenv(llgoWasiThreads, "")
	if IsWasiThreadsEnabled() {
		t.Fatal("WASI threads are enabled by default")
	}
	t.Setenv(llgoWasiThreads, "1")
	if !IsWasiThreadsEnabled() {
		t.Fatal("WASI threads opt-in was ignored")
	}
}

func TestCHeaderPackagesExcludesStandardRuntime(t *testing.T) {
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	userLPkg := prog.NewPackage("example.com/p", "example.com/p")
	userLPkg.SetExport("example.com/p.Export", "Export")
	runtimeLPkg := prog.NewPackage("runtime", "runtime")
	llgoRuntimeLPkg := prog.NewPackage("github.com/goplus/llgo/runtime/internal/lib/runtime", "github.com/goplus/llgo/runtime/internal/lib/runtime")
	dependencyLPkg := prog.NewPackage("example.com/dep", "example.com/dep")
	pkgs := []*aPackage{
		{Package: &packages.Package{PkgPath: "example.com/p"}, LPkg: userLPkg},
		{Package: &packages.Package{PkgPath: "runtime"}, LPkg: runtimeLPkg},
		{Package: &packages.Package{PkgPath: "github.com/goplus/llgo/runtime/internal/lib/runtime"}, LPkg: llgoRuntimeLPkg},
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
	if err := os.WriteFile(filepath.Join(runtimeDir, "go.mod"), []byte("module github.com/goplus/llgo/runtime\n\ngo 1.24.0\n"), 0o644); err != nil {
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
