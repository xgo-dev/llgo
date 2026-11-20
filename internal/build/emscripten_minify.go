package build

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/clang"
	"github.com/xgo-dev/llgo/internal/crosscompile"
)

const emccClosureArgs = "EMCC_CLOSURE_ARGS"
const closureSimpleArgs = "--compilation_level SIMPLE_OPTIMIZATIONS"

func shouldMinifyEmscriptenGlue(ctx *context, output string, env commandEnv, args []string) bool {
	conf := ctx.buildConf
	if ctx.skipEmscriptenMinifier || ctx.crossCompile.WasmProvider != crosscompile.WasmProviderEmscripten ||
		conf.DebugArtifactMode != DebugArtifactNone || conf.LinkOptions.ExternalLinkerFlags != "" || !isEmscriptenNamedTarget(conf) ||
		emscriptenBrowserHostExt(output) == "" || !hostedValueFormatSupport(conf, ctx.crossCompile, ctx.frontendOptions) {
		return false
	}
	for _, name := range []string{"LDFLAGS", "EMCC_CFLAGS", emccClosureArgs} {
		if env.lookup(name) != "" {
			return false
		}
	}
	return !slices.ContainsFunc(args, func(arg string) bool {
		return arg == "--closure" || strings.HasPrefix(arg, "--closure=") || strings.HasPrefix(arg, "@")
	})
}

func linkWithOptionalClosure(ctx *context, cmd *clang.Cmd, output string, args []string) error {
	// Archives carry the original toolchain configuration. Select the JS
	// optimization here, where the linker has complete Wasm and host glue.
	if !needsWasmFinalization(ctx.buildConf) {
		return cmd.Link(args...)
	}
	ctx.wasmJSMinifier = ""
	ctx.effectiveLinkArgs = cmd.LinkArguments(args...)
	environ := cmd.Env
	if environ == nil {
		environ = os.Environ()
	}
	if shouldMinifyEmscriptenGlue(ctx, output, commandEnv{environ: environ}, ctx.effectiveLinkArgs) {
		opt := *cmd
		opt.Env = withEnv(environ, emccClosureArgs+"="+closureSimpleArgs)
		extra := append(slices.Clone(args), "--closure", "1")
		var diagnostics bytes.Buffer
		opt.Stderr = &diagnostics
		err := opt.Link(extra...)
		if err == nil || !emscriptenClosureFailure(diagnostics.String()) {
			if cmd.Stderr != nil {
				_, _ = diagnostics.WriteTo(cmd.Stderr)
			}
			if err == nil {
				ctx.effectiveLinkArgs = opt.LinkArguments(extra...)
				ctx.wasmJSMinifier = "closure-simple"
			}
			return err
		}
		// A usable Emscripten installation may omit the optional npm/Java
		// Closure tool. Retry the driver with the original archives and flags.
		ctx.skipEmscriptenMinifier = true
		if cmd.Verbose {
			fmt.Fprintln(os.Stderr, "llgo: Emscripten JavaScript minification failed; retrying with the default linker settings")
			_, _ = io.Copy(os.Stderr, &diagnostics)
		}
	}
	return cmd.Link(args...)
}

func emscriptenClosureFailure(output string) bool {
	return strings.Contains(output, "closure compiler failed") ||
		strings.Contains(output, "google-closure-compiler was not found") ||
		strings.Contains(output, "unrecognized closure compiler --version output") ||
		(strings.Contains(output, "closure compiler (") && strings.Contains(output, "did not execute properly"))
}
