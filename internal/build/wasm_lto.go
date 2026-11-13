package build

import "github.com/xgo-dev/llgo/internal/crosscompile"

func emscriptenLTOLibcallArgs(ctx *context) []string {
	profile := ctx.crossCompile.WasmProfile
	if !ctx.buildConf.ltoEnabled() || (profile != crosscompile.WasmProfileJ32 && profile != crosscompile.WasmProfileJ64) {
		return nil
	}
	// Emscripten 6.0.8 keeps fputc in a native archive member, since LTO can
	// introduce it when simplifying fwrite. Its thread helpers remain bitcode,
	// which wasm-ld cannot extract after LTO has introduced that native member.
	// Resolve them before LTO, using the SDK's stubs or real pthread versions
	// according to the selected target. Both Wasm pointer widths are affected.
	return []string{"-Wl,--undefined=__get_tp", "-Wl,--undefined=emscripten_futex_wake"}
}
