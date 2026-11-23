package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
)

func TestEmscriptenSchedulerProofProfile(t *testing.T) {
	conf := Config{Mode: ModeBuild, BuildMode: BuildModeExe, Target: "emscripten",
		Goos: "js", Goarch: "wasm", OptLevel: optlevel.Oz, LTO: lto.Full,
		DeadcodeDrop: true, DisableGoGlobalDCE: true, DebugArtifactMode: DebugArtifactNone, WasmOpt: WasmOptSize}
	if !conf.deadcodeDropEnabled() {
		t.Skip("development DCE is unavailable")
	}
	target := crosscompile.Export{LDFLAGS: []string{"-sASYNCIFY=1"}}
	if !emscriptenSchedulerProofProfile(&conf, &target) {
		t.Fatal("single-worker size executable did not enable the proof")
	}
	workers := target
	workers.BuildTags = []string{"llgo.wasm.workers"}
	if emscriptenSchedulerProofProfile(&conf, &workers) {
		t.Fatal("worker build enabled the proof")
	}
	conf.LTO = lto.Off
	if emscriptenSchedulerProofProfile(&conf, &target) {
		t.Fatal("ordinary DCE build enabled the scheduler probe")
	}
	conf.WasmMinimalHost = true
	if !emscriptenSchedulerProofProfile(&conf, &target) {
		t.Fatal("opt-in DCE build did not enable the scheduler probe")
	}
	conf.WasmOpt = WasmOptOff
	if emscriptenSchedulerProofProfile(&conf, &target) {
		t.Fatal("build without staged Wasm finalization enabled the proof")
	}
}

func TestEmscriptenSchedulerProofPackage(t *testing.T) {
	for _, directive := range []string{"", "//export f", "//go:wasmexport f", "//go:wasmimport env f", "//go:linkname f p.f", "//llgo:export f", "//llgo:link f C.f"} {
		t.Run(directive, func(t *testing.T) {
			src, err := parser.ParseFile(token.NewFileSet(), "p.go", "package p\n"+directive+"\nfunc f() {}", parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			pkg := &packages.Package{Name: "p", Dir: t.TempDir(), Syntax: []*ast.File{src}}
			if got := wasmSchedulerProbePackage(pkg, ""); got != (directive == "") {
				t.Fatalf("package accepted = %t", got)
			}
		})
	}
	pkg := &packages.Package{Name: "p", Dir: t.TempDir(), OtherFiles: []string{"bridge.c"}}
	if wasmSchedulerProbePackage(pkg, "") {
		t.Fatal("accepted an external object")
	}
	pkg.OtherFiles = nil
	if err := os.Mkdir(filepath.Join(pkg.Dir, "_wrap"), 0o700); err != nil {
		t.Fatal(err)
	}
	if wasmSchedulerProbePackage(pkg, "") {
		t.Fatal("accepted a wrapper directory")
	}
}

func TestEmscriptenSchedulerLinkFallback(t *testing.T) {
	for _, mode := range []string{"write-wasm", "reject-direct-wasm", "fail"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("LLGO_TEST_FINAL_WASM", "1")
			t.Setenv("LLGO_TEST_LINKER_HELPER", mode)
			dir := t.TempDir()
			ctx := &context{
				buildConf: &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm",
					Target: "emscripten", WasmStrip: WasmStripAll, DebugArtifactMode: DebugArtifactNone, PCLNMode: PCLNNone},
				crossCompile: crosscompile.Export{
					WasmProvider: crosscompile.WasmProviderEmscripten, CC: os.Args[0],
					LDFLAGS: []string{"-sASYNCIFY=1"},
				},
			}
			// Use an extension without browser sidecars to exercise only link selection.
			name := "app"
			plan := &mainLinkPlan{linkInputs: []string{"normal.o"}, directWasmInputs: []string{"direct.o"}}
			result, stage, err := linkStagedWasm(ctx, plan, name, dir, true)
			if mode == "fail" {
				if err == nil {
					t.Fatal("normal link error was swallowed")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(stage, name)); err != nil {
				t.Fatal(err)
			}
			direct := mode == "write-wasm"
			if (result != ctx) != direct || (stage != dir) != direct ||
				slices.Contains(result.effectiveLinkArgs, "-sASYNCIFY=1") == direct ||
				slices.Contains(result.effectiveLinkArgs, "-sERROR_ON_UNDEFINED_SYMBOLS=1") != direct {
				t.Fatalf("stage %q selected with %q", stage, result.effectiveLinkArgs)
			}
			if !direct {
				files, err := os.ReadDir(dir)
				if err != nil {
					t.Fatal(err)
				}
				if len(files) != 1 || files[0].Name() != name {
					t.Fatalf("failed candidate left files in staging: %v", files)
				}
			}
		})
	}
}

func TestEmscriptenSchedulerStageCreationError(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	ctx := &context{}
	plan := &mainLinkPlan{directWasmInputs: []string{"direct.o"}}
	if out, stage, err := linkStagedWasm(ctx, plan, "app", dir, false); !os.IsNotExist(err) || out != ctx || stage != dir {
		t.Fatalf("unexpected staging result: %p %q %v", out, stage, err)
	}
}

func TestEmscriptenSchedulerCollectorFallback(t *testing.T) {
	t.Setenv("LLGO_TEST_FINAL_WASM", "1")
	t.Setenv("LLGO_TEST_LINKER_HELPER", "reject-direct-wasm")
	dir := t.TempDir()
	ctx := &context{
		buildConf: &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm",
			Target: "emscripten", WasmStrip: WasmStripAll, DebugArtifactMode: DebugArtifactNone, PCLNMode: PCLNNone},
		crossCompile: crosscompile.Export{
			WasmProvider: crosscompile.WasmProviderEmscripten, CC: os.Args[0],
			LDFLAGS: []string{"-sMALLOC=none", emscriptenBrowserFSRoot},
		},
	}
	plan := &mainLinkPlan{
		linkInputs: []string{"normal.o"}, directWasmInputs: []string{"direct.o"}, directWasmAllocator: true,
	}
	result, stage, err := linkStagedWasm(ctx, plan, "app", dir, true)
	if err != nil || result != ctx || stage != dir {
		t.Fatalf("collector fallback failed: %p %q %v", result, stage, err)
	}
	if !slices.Contains(result.effectiveLinkArgs, "-sMALLOC=none") ||
		slices.Contains(result.effectiveLinkArgs, "-sMALLOC=emmalloc") ||
		!slices.Contains(result.effectiveLinkArgs, emscriptenBrowserFSRoot) || result.omitEmscriptenBrowserHost {
		t.Fatalf("fallback allocator flags = %q", result.effectiveLinkArgs)
	}
}

func TestEmscriptenSchedulerLibcAllocatorFlags(t *testing.T) {
	shared := []string{"--pre-js", "foo.js", "-sEXPORTED_RUNTIME_METHODS=FS"}
	browserHost := []string{"--pre-js", wasmFSScriptName, "--js-library", "browser_fs.js",
		"--js-library", "browser_fs_snapshot.js", emscriptenBrowserFSRoot}
	flags := slices.Concat([]string{"-sASYNCIFY=1", "-sMALLOC=none"}, shared, browserHost)
	ctx := &context{emscriptenDirectMain: true, crossCompile: crosscompile.Export{
		LDFLAGS: slices.Clone(flags),
	}}
	normal := newLinkExecutionContext(ctx, &mainLinkPlan{})
	libc := newLinkExecutionContext(ctx, &mainLinkPlan{emscriptenUseLibcAllocator: true})
	if !slices.Equal(normal.crossCompile.LDFLAGS, flags[1:]) || normal.omitEmscriptenBrowserHost ||
		!slices.Equal(libc.crossCompile.LDFLAGS, slices.Concat([]string{"-sMALLOC=emmalloc"}, shared)) ||
		!libc.omitEmscriptenBrowserHost || !slices.Equal(ctx.crossCompile.LDFLAGS, flags) {
		t.Fatal("allocator selection changed the profile or selected the wrong link flags")
	}
}

func TestEmscriptenSchedulerEntryPackageFingerprint(t *testing.T) {
	section := packageSection{EmscriptenRuntimeFree: true}
	if section.empty() {
		t.Fatal("allocation-free entry must participate in the archive fingerprint")
	}
}

func TestEmscriptenMinimalHostLink(t *testing.T) {
	t.Setenv("LLGO_TEST_FINAL_WASM", "1")
	t.Setenv("LLGO_TEST_LINKER_HELPER", "write-wasm")
	ctx := &context{
		buildConf: &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm",
			Target: "emscripten", WasmStrip: WasmStripAll, DebugArtifactMode: DebugArtifactNone, PCLNMode: PCLNNone},
		crossCompile: crosscompile.Export{
			WasmProvider: crosscompile.WasmProviderEmscripten, CC: os.Args[0],
			LDFLAGS: []string{"-sASYNCIFY=1", "-sEXPORTED_RUNTIME_METHODS=FS,ENV",
				"-sMALLOC=none", "--js-library", "browser_fs.js", emscriptenBrowserFSRoot},
		},
		emscriptenMinimalHost: true,
	}
	plan := &mainLinkPlan{linkInputs: []string{"normal.o"}, directWasmInputs: []string{"direct.o"}}
	result, _, err := linkStagedWasm(ctx, plan, "app", t.TempDir(), true)
	if err != nil {
		t.Fatal(err)
	}
	if !result.omitEmscriptenBrowserHost ||
		!slices.Contains(result.effectiveLinkArgs, "-sFILESYSTEM=0") ||
		!slices.Contains(result.effectiveLinkArgs, "-sMALLOC=none") ||
		slices.Contains(result.effectiveLinkArgs, "-sASYNCIFY=1") ||
		slices.Contains(result.effectiveLinkArgs, "-sEXPORTED_RUNTIME_METHODS=FS,ENV") ||
		slices.Contains(result.effectiveLinkArgs, "browser_fs.js") {
		t.Fatalf("unexpected minimal host flags: %q", result.effectiveLinkArgs)
	}
}
