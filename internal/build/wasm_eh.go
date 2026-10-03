package build

import (
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llvm"
)

// Emscripten can select native Wasm SjLj through SUPPORT_LONGJMP=wasm. LLVM
// requires the feature on each caller's IR function, even when emcc receives
// -fwasm-exceptions. Declaring the capability does not select the EH mode:
// Emscripten's default JS SjLj and explicit native SjLj keep their own lowering.
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
