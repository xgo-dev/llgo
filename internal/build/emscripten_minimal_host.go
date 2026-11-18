package build

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/env"
)

// canUseMinimalEmscriptenHost checks configuration that can introduce host
// dependencies outside the Go SSA proof.
func canUseMinimalEmscriptenHost(ctx *context) bool {
	conf := ctx.buildConf
	if !conf.WasmMinimalHost || ctx.crossCompile.WasmProvider != crosscompile.WasmProviderEmscripten ||
		ctx.crossCompile.WasmRuntime.RunMainTask || conf.LinkOptions.ExternalLinker != "" ||
		conf.DebugArtifactMode == DebugArtifactEmbedded || conf.DebugArtifactMode == DebugArtifactExternal ||
		conf.ModuleHook != nil || len(conf.SourceOverlay) != 0 {
		return false
	}
	tags := append(splitSourcePatchBuildTags(conf.Tags), ctx.crossCompile.BuildTags...)
	if slices.Contains(tags, "llgo.wasm.workers") || slices.Contains(tags, wasmReflectLibffiTag) {
		return false
	}
	for _, key := range []string{"CCFLAGS", "CFLAGS", "EMCC_CFLAGS"} {
		if os.Getenv(key) != "" || ctx.commands.lookup(key) != "" {
			return false
		}
	}
	return true
}

// selectMinimalEmscriptenHost is called after proving that neither Go nor
// native inputs need allocation, host callbacks, or scheduling. Keep the
// ordinary Emscripten driver and its standard executable entry.
func selectMinimalEmscriptenHost(ctx *context) {
	conf := ctx.buildConf
	conf.wasmMinimalHostSelected = true
	ctx.prog.EnableDeadcodeDrop(conf.deadcodeDropEnabled())
	root := env.LLGoROOT()
	fsShim := filepath.Join(root, "targets", wasmFSScriptName)
	asyncifyShim := filepath.Join(root, "targets", "emscripten-asyncify-exports.js")
	browserFS := filepath.Join(root, "internal", "wasmworkers", "browser_fs.js")

	var kept []string
	flags := ctx.crossCompile.LDFLAGS
	for i := 0; i < len(flags); i++ {
		arg := flags[i]
		if i+1 < len(flags) && ((arg == "--pre-js" && (flags[i+1] == fsShim || flags[i+1] == asyncifyShim)) ||
			(arg == "--js-library" && flags[i+1] == browserFS)) {
			i++
			continue
		}
		if arg == "-sASYNCIFY=1" || strings.HasPrefix(arg, "-sASYNCIFY_IMPORTS=") ||
			strings.HasPrefix(arg, "-sASYNCIFY_REMOVE=") || strings.HasPrefix(arg, "-sEXPORTED_RUNTIME_METHODS=") ||
			arg == "-sDEFAULT_LIBRARY_FUNCS_TO_INCLUDE=$llgoBrowserFS" {
			continue
		}
		kept = append(kept, arg)
	}
	ctx.crossCompile.LDFLAGS = append(kept, "-sFILESYSTEM=0", "-sEXPORTED_RUNTIME_METHODS=ENV", "--closure", "1")
	ctx.crossCompile.CCFLAGS = append(slices.Clone(ctx.crossCompile.CCFLAGS), "-DLLGO_EMSCRIPTEN_NO_ASYNCIFY")
	ctx.crossCompile.WasmPostLink.Asyncify = false
	if conf.Verbose || conf.PrintCommands {
		fmt.Fprintln(os.Stderr, "llgo: select minimal Emscripten host")
	}
}
