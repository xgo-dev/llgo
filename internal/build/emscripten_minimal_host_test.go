package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestMinimalEmscriptenHost(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	for _, key := range []string{"LDFLAGS", "CCFLAGS", "CFLAGS", "EMCC_CFLAGS", llgoWasmWorkers} {
		t.Setenv(key, "")
	}
	for _, tc := range []struct {
		name   string
		source string
		change func(*testing.T, *context, *aPackage)
		want   bool
		usesGC bool
	}{
		{name: "eligible", want: true},
		{name: "emit diagnostic", want: true, change: func(_ *testing.T, ctx *context, _ *aPackage) {
			ctx.buildConf.PrintCommands = true
		}},
		{name: "cached minimal host", want: true, change: func(_ *testing.T, ctx *context, _ *aPackage) {
			ctx.frontend = &frontendCache{program: &frontendProgramExport{RuntimeFree: true, MinimalHost: true}}
		}},
		{name: "cached synchronous host", want: true, usesGC: true, change: func(_ *testing.T, ctx *context, _ *aPackage) {
			ctx.frontend = &frontendCache{program: &frontendProgramExport{MinimalHost: true}}
		}},
		{name: "cached fallback", change: func(_ *testing.T, ctx *context, _ *aPackage) {
			ctx.frontend = &frontendCache{program: &frontendProgramExport{}}
		}},
		{name: "default host", change: func(_ *testing.T, ctx *context, _ *aPackage) {
			ctx.buildConf.WasmMinimalHost = false
		}},
		{name: "panic", source: "func main() { panic(42) }"},
		{name: "synchronous allocation", change: func(_ *testing.T, ctx *context, _ *aPackage) {
			ctx.buildConf.PrintCommands = true
		}, source: "func main(){ s:=make([]string,1); s[0]=\"hi\"; println(s[0]) }", want: true, usesGC: true},
		{name: "channel operation", source: "func wait(ch chan int){ println(<-ch) }; func main(){ wait(nil) }"},
		{name: "interface print", source: "func main(){ println(any(42)) }"},
		{name: "unknown builtin", source: "func main(){ println(recover() != nil) }"},
		{name: "recursive synchronous calls", source: "func f(n int){ if n > 0 { f(n-1) }; println(len(make([]int,n))) }; func main(){ f(3) }", want: true, usesGC: true},
		{name: "linked global", source: "var a = []int{1}; func main(){ println(a[0]) }", change: func(_ *testing.T, ctx *context, _ *aPackage) {
			ctx.prog.SetLinkname("main.a", "C.a")
		}},
		{name: "channel receive", source: "func main(){ ch:=make(chan int); println(<-ch) }"},
		{name: "deferred call", source: "func emit(){ println(0) }; func main(){ defer emit() }"},
		{name: "indirect call", source: "var f = func() int { return 7 }; func main(){ println(f()) }"},
		{name: "patched package", source: "func main(){ println(len(make([]int,1))) }", change: func(_ *testing.T, ctx *context, p *aPackage) {
			ctx.patches = cl.Patches{p.Types.Path(): {Alt: p.SSA}}
		}},

		{name: "goroutine", source: "func main() { go func() { println(42) }() }"},
		{name: "workers", change: func(_ *testing.T, ctx *context, _ *aPackage) {
			ctx.crossCompile.WasmRuntime.RunMainTask = true
		}},
		{name: "libffi", change: func(_ *testing.T, ctx *context, _ *aPackage) {
			ctx.crossCompile.BuildTags = []string{wasmReflectLibffiTag}
		}},
		{name: "external linker", change: func(_ *testing.T, ctx *context, _ *aPackage) {
			ctx.buildConf.LinkOptions.ExternalLinker = "custom-emcc"
		}},
		{name: "compiler environment", change: func(t *testing.T, _ *context, _ *aPackage) {
			t.Setenv("EMCC_CFLAGS", "--pre-js extra.js")
		}},
		{name: "debug artifact", change: func(_ *testing.T, ctx *context, _ *aPackage) {
			ctx.buildConf.DebugArtifactMode = DebugArtifactEmbedded
		}},
		{name: "source overlay", change: func(_ *testing.T, ctx *context, _ *aPackage) {
			ctx.buildConf.SourceOverlay = map[string]SourceOverlayFile{"file.go": {}}
		}},
		{name: "module hook", change: func(_ *testing.T, ctx *context, _ *aPackage) {
			ctx.buildConf.ModuleHook = func(Package) {}
		}},
		{name: "foreign export", change: func(_ *testing.T, _ *context, p *aPackage) {
			p.Syntax[0].Comments = []*ast.CommentGroup{{List: []*ast.Comment{{Text: "//export callback"}}}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			contents := tc.source
			if contents == "" {
				contents = "func main() { println(42) }"
			}
			src := "package main\n" + contents
			ssp := buildWasmReflectTestProgram(t, src)
			f, err := parser.ParseFile(token.NewFileSet(), "main.go", src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			p := &aPackage{Package: &packages.Package{ID: "main", PkgPath: "main", Dir: t.TempDir(),
				Types: ssp.Pkg, Syntax: []*ast.File{f}}, SSA: ssp}
			prog := llssa.NewProgram(&llssa.Target{GOOS: "js", GOARCH: "wasm", WasmProvider: "emscripten"})
			defer prog.Dispose()
			prog.EnableGCRoots(true)
			prog.EnableCooperativeSafepoints(true)
			conf := &Config{Goos: "js", Goarch: "wasm", Target: "emscripten", OptLevel: optlevel.Oz,
				BuildMode: BuildModeExe, WasmMinimalHost: true, DebugArtifactMode: DebugArtifactNone}
			target := crosscompile.Export{WasmProvider: crosscompile.WasmProviderEmscripten, WasmProfile: crosscompile.WasmProfileJ32,
				LDFLAGS: []string{"-sASYNCIFY=1", "-sASYNCIFY_IMPORTS=wait", "-sASYNCIFY_REMOVE=neverWait",
					"-sEXPORTED_RUNTIME_METHODS=cwrap,FS,ENV", "-sSTACK_OVERFLOW_CHECK=2"}}
			if _, err := configureWasmWorkers(conf, &target); err != nil {
				t.Fatal(err)
			}
			ctx := &context{buildConf: conf, mode: ModeBuild, prog: prog, progSSA: ssp.Prog, crossCompile: target,
				initial: []*packages.Package{p.Package}, pkgByID: map[string]Package{p.ID: p}}
			if tc.change != nil {
				tc.change(t, ctx, p)
			}
			ctx.frontendOptions.ShadowStack = true
			if ctx.buildConf.PrintCommands {
				finish := captureStderr(t)
				configureWasmRuntimeFree(ctx)
				got := finish()
				if !strings.Contains(got, "select minimal Emscripten host") ||
					tc.usesGC && !strings.Contains(got, "keep GC with a synchronous Emscripten host") {
					t.Fatalf("missing minimal host diagnostic: %s", got)
				}
			} else {
				configureWasmRuntimeFree(ctx)
			}
			if conf.wasmMinimalHostSelected != tc.want || ctx.wasmRuntimeFree != (tc.want && !tc.usesGC) || conf.deadcodeDropEnabled() != tc.want {
				t.Fatalf("minimal=%v runtime-free=%v DCE=%v, want %v", conf.wasmMinimalHostSelected, ctx.wasmRuntimeFree, conf.deadcodeDropEnabled(), tc.want)
			}
			if prog.GCRootsEnabled() != !ctx.wasmRuntimeFree || ctx.frontendOptions.ShadowStack != !ctx.wasmRuntimeFree ||
				prog.CooperativeSafepointsEnabled() == tc.want {
				t.Fatal("GC roots, shadow stack or scheduling polls do not match host policy")
			}
			inputs := newManifestBuilder()
			ctx.collectCommonInputs(inputs)
			if inputs.common.WasmSynchronousGC != tc.usesGC {
				t.Fatal("GC-capable synchronous builds must have a distinct archive fingerprint")
			}
			if needsWasmRuntimeScheduler(ctx) == tc.want || needsEmscriptenBrowserHost(conf, "app.mjs") == tc.want {
				t.Fatal("scheduler and sidecar must match the resolved host")
			}
			if tc.want {
				want := []string{"-sSTACK_OVERFLOW_CHECK=2", "-sFILESYSTEM=0", "-sEXPORTED_RUNTIME_METHODS=ENV", "--closure", "1"}
				if !slices.Equal(ctx.crossCompile.LDFLAGS, want) {
					t.Fatalf("link flags = %v, want %v", ctx.crossCompile.LDFLAGS, want)
				}
				if !slices.Contains(ctx.crossCompile.CCFLAGS, "-DLLGO_EMSCRIPTEN_NO_ASYNCIFY") {
					t.Fatal("C wrapper still references Asyncify")
				}
				if conf.clone().wasmMinimalHostSelected {
					t.Fatal("a clone retained the previous program's host selection")
				}
			}
		})
	}
}

func TestValidateMinimalEmscriptenHost(t *testing.T) {
	for _, tc := range []struct {
		name    string
		target  string
		profile crosscompile.WasmProfile
		host    crosscompile.WasmProvider
		mode    BuildMode
		want    bool
	}{
		{"emscripten", "emscripten", crosscompile.WasmProfileJ32, crosscompile.WasmProviderEmscripten, BuildModeExe, true},
		{"memory64", "emscripten-memory64", crosscompile.WasmProfileJ64, crosscompile.WasmProviderEmscripten, BuildModeExe, true},
		{"archive", "emscripten", crosscompile.WasmProfileJ32, crosscompile.WasmProviderEmscripten, BuildModeCArchive, false},
		{"WASI", "wasi", crosscompile.WasmProfileW32, crosscompile.WasmProviderWASI, BuildModeExe, false},
		{"raw js/wasm", "", crosscompile.WasmProfileJ32, crosscompile.WasmProviderGoJS, BuildModeExe, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conf := &Config{Target: tc.target, Mode: ModeBuild, BuildMode: tc.mode, WasmMinimalHost: true}
			target := &crosscompile.Export{WasmProfile: tc.profile, WasmProvider: tc.host}
			if err := validateWasmFinalOptions(conf, target); (err == nil) != tc.want {
				t.Fatalf("validate -wasm-minimal-host: %v, want success=%v", err, tc.want)
			}
		})
	}
}

func TestSynchronousWasmNeedsExecutableRoots(t *testing.T) {
	if wasmSynchronousFunctions(nil, nil, nil) {
		t.Fatal("an entry-less package cannot prove synchronous execution")
	}
}
