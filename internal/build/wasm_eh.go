package build

import (
	"fmt"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/xtool/safesplit"
	"github.com/xgo-dev/llvm"
)

// Emscripten uses native Wasm SjLj through SUPPORT_LONGJMP=wasm. LLVM
// requires the feature on each caller's IR function, even when emcc receives
// -fwasm-exceptions. The cross-compilation profile selects native SjLj;
// function attributes must retain the capability through optimization and LTO.
func applyEmscriptenEHFeature(ctx *context, mod llvm.Module) {
	profile := ctx.crossCompile.WasmProfile
	if profile != crosscompile.WasmProfileJ32 && profile != crosscompile.WasmProfileJ64 {
		return
	}
	required := []string{"exception-handling"}
	if slices.Contains(ctx.crossCompile.CCFLAGS, "-pthread") {
		// Function attributes override the backend's command-line features.
		// Preserve the capabilities selected by emcc -pthread: shared-memory
		// objects require both, and TLS accesses require atomics lowering.
		required = append(required, "atomics", "bulk-memory")
	}
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if fn.IsDeclaration() {
			continue
		}
		features := make([]string, 0, len(required))
		for _, feature := range required {
			features = append(features, "+"+feature)
		}
		for _, attr := range fn.GetFunctionAttributes() {
			if attr.IsString() && attr.GetStringKind() == "target-features" {
				for _, feature := range strings.Split(attr.GetStringValue(), ",") {
					if feature != "" && !slices.Contains(required, strings.TrimLeft(feature, "+-")) {
						features = append(features, feature)
					}
				}
			}
		}
		fn.AddFunctionAttr(mod.Context().CreateStringAttribute("target-features", strings.Join(features, ",")))
	}
}

// Go panic/recover and the runtime's C helpers must use the same SjLj ABI.
// EMCC_CFLAGS is appended by emcc after our profile flags, so reject attempts
// to restore JavaScript SjLj rather than silently compiling incompatible code.
func validateEmscriptenSjLj(commands commandEnv, target *crosscompile.Export) error {
	if target.WasmProvider != crosscompile.WasmProviderGoJS && target.WasmProvider != crosscompile.WasmProviderEmscripten {
		return nil
	}
	for _, name := range []string{"CCFLAGS", "CFLAGS", "LDFLAGS", "EMCC_CFLAGS"} {
		if err := validateEmscriptenEHArgs(name, safesplit.SplitPkgConfigFlags(commands.lookup(name))); err != nil {
			return err
		}
	}
	return nil
}

// The same checks apply to package link directives, which are appended after
// the profile flags.
func validateEmscriptenEHArgs(name string, args []string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-s" && i+1 < len(args) {
			i++
			arg += args[i]
		}
		value, setting := strings.CutPrefix(arg, "-sSUPPORT_LONGJMP=")
		if setting && strings.Trim(value, "\"'") != "wasm" || arg == "-sSUPPORT_LONGJMP" || strings.HasPrefix(arg, "-enable-emscripten-sjlj") {
			return fmt.Errorf("%s contains %q: LLGo requires -sSUPPORT_LONGJMP=wasm for Go panic/recover", name, arg)
		}
		if strings.Contains(arg, "asyncify-ignore-unwind-from-catch") {
			return fmt.Errorf("%s contains %q: LLGo cannot ignore unsupported Asyncify suspension inside a Wasm catch", name, arg)
		}
	}
	return nil
}
