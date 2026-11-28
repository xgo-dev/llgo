package build

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xgo-dev/llgo/internal/clang"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
)

func TestEmscriptenMinifierDriver(t *testing.T) {
	if mode := os.Getenv("LLGO_TEST_CLOSURE_DRIVER"); mode != "" {
		closure := slices.Contains(os.Args, "--closure")
		if closure && os.Getenv(emccClosureArgs) != closureSimpleArgs {
			fmt.Fprintln(os.Stderr, "missing SIMPLE optimization")
			os.Exit(2)
		}
		if closure && mode == "missing" {
			fmt.Fprintln(os.Stderr, "emcc: error: google-closure-compiler was not found!")
			os.Exit(1)
		}
		if mode == "link-error" {
			fmt.Fprintln(os.Stderr, "wasm-ld: error: undefined symbol")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, closure)
		os.Exit(0)
	}

	conf := &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Target: "emscripten",
		Goos: "js", Goarch: "wasm", OptLevel: optlevel.Oz, LTO: lto.Full,
		DeadcodeDrop: true, DisableGoGlobalDCE: true, DebugArtifactMode: DebugArtifactNone, WasmOpt: WasmOptSize}
	if !conf.deadcodeDropEnabled() {
		t.Skip("development DCE is unavailable")
	}
	t.Setenv("CCFLAGS", "")
	t.Setenv("LDFLAGS", "")
	for _, tt := range []struct {
		name        string
		mode        string
		override    []string
		environment []string
		minifier    string
		want        string
	}{
		{name: "success", mode: "success", minifier: "closure-simple", want: "true\n"},
		{name: "fallback", mode: "missing", want: "false\n"},
		{name: "link-error", mode: "link-error"},
		{name: "user setting", mode: "success", override: []string{"--closure=0"}, want: "false\n"},
		{name: "user environment", mode: "success", environment: []string{emccClosureArgs + "=--debug"}, want: "false\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &context{buildConf: conf, crossCompile: crosscompile.Export{
				WasmProfile: crosscompile.WasmProfileJ32, WasmProvider: crosscompile.WasmProviderEmscripten,
			}}
			target := filepath.Join(t.TempDir(), "app.mjs")
			args := append([]string{"-o", target}, tt.override...)
			cmd := clang.NewLinker(clang.Config{Linker: os.Args[0], LinkerArgs: []string{"-test.run=^TestEmscriptenMinifierDriver$", "--"}})
			cmd.Env = withEnv(os.Environ(), "LLGO_TEST_CLOSURE_DRIVER="+tt.mode,
				"LDFLAGS=", "EMCC_CFLAGS=", emccClosureArgs+"=")
			cmd.Env = withEnv(cmd.Env, tt.environment...)
			var out, diagnostics bytes.Buffer
			cmd.Stdout, cmd.Stderr = &out, &diagnostics
			err := linkWithOptionalClosure(ctx, cmd, target, args)
			if tt.mode == "link-error" {
				if err == nil || diagnostics.String() != "wasm-ld: error: undefined symbol\n" || ctx.skipEmscriptenMinifier {
					t.Fatalf("ordinary linker error changed: %v; %s", err, &diagnostics)
				}
				return
			}
			if err != nil || out.String() != tt.want || diagnostics.Len() != 0 {
				t.Fatalf("link result: %v, stdout=%q, stderr=%q", err, out.String(), diagnostics.String())
			}
			if ctx.wasmJSMinifier != tt.minifier || slices.Contains(ctx.effectiveLinkArgs, "--closure") != (tt.minifier != "") {
				t.Fatalf("unexpected manifest state: %q, %q", ctx.wasmJSMinifier, ctx.effectiveLinkArgs)
			}
			next := newLinkExecutionContext(ctx, &mainLinkPlan{})
			if next.skipEmscriptenMinifier != (tt.mode == "missing") {
				t.Fatal("retry policy did not persist to subsequent links")
			}
		})
	}
}
