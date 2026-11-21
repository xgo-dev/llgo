package build

import (
	"bytes"
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

const (
	emscriptenSchedulerProofTag = "llgo.wasm.schedulerproof"
	emscriptenSchedulerMarker   = "__llgo_wasm_requires_scheduler"
	emscriptenMainDispatch      = "__llgo_wasm_run_main"
	wasmMainTaskSymbol          = "__llgo_wasm_main"
)

// Compile the same runtime for every executable in this profile. The link
// supplies the marker when it supplies the normal scheduler.
func emscriptenSchedulerProofProfile(conf *Config, target *crosscompile.Export) bool {
	return isEmscriptenNamedTarget(conf) && sizeCodecBuild(conf, cl.Options{}) &&
		conf.DebugArtifactMode == DebugArtifactNone && needsWasmFinalization(conf) && !conf.goGlobalDCEEnabled() &&
		slices.Contains(target.LDFLAGS, "-sASYNCIFY=1") &&
		!slices.Contains(target.BuildTags, "llgo.wasm.workers")
}

func emscriptenSchedulerProofEnabled(ctx *context) bool {
	return slices.Contains(ctx.crossCompile.BuildTags, emscriptenSchedulerProofTag) ||
		slices.Contains(splitSourcePatchBuildTags(ctx.buildConf.Tags), emscriptenSchedulerProofTag)
}

func canProbeEmscriptenScheduler(ctx *context, inputs *linkPruningInputs) bool {
	conf := ctx.buildConf
	if !slices.Contains(ctx.crossCompile.BuildTags, emscriptenSchedulerProofTag) ||
		ctx.emscriptenDirectMain || !canUseEmscriptenDirectMain(ctx) ||
		conf.PCLNMode == PCLNExternal || conf.GenLL || conf.CheckLLFiles ||
		len(ctx.initial) != 1 || !ctx.wasmClosedProgram || inputs == nil || inputs.opaqueAsmPkg != "" ||
		conf.LinkOptions.ExternalLinkerFlags != "" || os.Getenv("LDFLAGS") != "" || ctx.commands.lookup("LDFLAGS") != "" {
		return false
	}
	// Unknown host entries can reenter Go, and foreign objects can suspend
	// without going through the marked runtime operations.
	for _, pkg := range ctx.pkgByID {
		if !wasmSchedulerProbePackage(pkg.Package, inputs.goroot) ||
			pkg.AltPkg != nil && !wasmSchedulerProbePackage(pkg.AltPkg.Package, inputs.goroot) {
			return false
		}
	}
	return true
}

func wasmSchedulerProbePackage(pkg *packages.Package, goroot string) bool {
	if isStandardLibraryPackage(pkg, goroot) || isSchedulerRuntimePackage(pkg) {
		return true
	}
	if len(pkg.OtherFiles) != 0 || cl.IsCPackage(pkg.Name) {
		return false
	}
	if _, err := os.Stat(filepath.Join(pkg.Dir, "_wrap")); !os.IsNotExist(err) {
		return false
	}
	for _, file := range pkg.Syntax {
		for _, group := range file.Comments {
			for _, item := range directive.ParseGroup(group) {
				switch item.Name {
				case "export", "llgo:export", "go:wasmexport", "go:wasmimport", "go:linkname", "llgo:link":
					return false
				}
			}
		}
	}
	return true
}

func isSchedulerRuntimePackage(pkg *packages.Package) bool {
	if pkg.Module == nil || pkg.Module.Path != env.LLGoRuntimePkg {
		return false
	}
	root, err := filepath.EvalSymlinks(env.LLGoRuntimeDir())
	if err != nil || root == "" {
		return false
	}
	dir, err := filepath.EvalSymlinks(pkg.Dir)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, dir)
	return err == nil && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func defineEmscriptenSchedulerMarker(pkg llssa.Package) {
	fn := pkg.NewFunc(emscriptenSchedulerMarker, llssa.NoArgsNoRet, llssa.CDecl)
	pkg.Module().NamedFunction(emscriptenSchedulerMarker).SetVisibility(llvm.HiddenVisibility)
	fn.MakeBody(1).Return()
}

// Both links use the same main object and archives. Only the dispatch object
// differs. The direct variant leaves the scheduler marker unresolved, so
// the linker must discard every path that can schedule or suspend.
func buildEmscriptenDispatch(ctx *context, exportFile string) (normal, direct string, err error) {
	makeDispatch := func(file string, direct bool) (string, error) {
		pkg := ctx.prog.NewPackage("", "emscripten.dispatch")
		defer pkg.Module().Dispose()
		f := pkg.NewFunc(emscriptenMainDispatch, llssa.NoArgsNoRet, llssa.CDecl)
		pkg.Module().NamedFunction(emscriptenMainDispatch).SetVisibility(llvm.HiddenVisibility)
		b := f.MakeBody(1)
		if direct {
			ptr := types.Typ[types.UnsafePointer]
			task := pkg.NewFunc(wasmMainTaskSymbol, newSignature([]types.Type{ptr}, []types.Type{ptr}), llssa.CDecl)
			b.Call(task.Expr, ctx.prog.Nil(ctx.prog.VoidPtr()))
		} else {
			defineEmscriptenSchedulerMarker(pkg)
			b.Call(declareNoArgFunc(pkg, llssa.PkgRuntime+".RunWasmMain").Expr)
		}
		b.Return()
		return exportObject(ctx, "emscripten_dispatch", file, pkg)
	}
	normal, err = makeDispatch(exportFile+"-scheduler", false)
	if err != nil {
		return "", "", err
	}
	direct, err = makeDispatch(exportFile+"-direct", true)
	return
}

// Keep failed speculative products out of the normal staging directory.
// Only the selected link contributes its flags and outputs to the manifest.
func linkStagedWasm(ctx *context, plan *mainLinkPlan, output, dir string, verbose bool) (*context, string, error) {
	local := *plan
	local.outputPath = filepath.Join(dir, filepath.Base(output))
	if len(plan.directWasmInputs) != 0 {
		probeDir, err := os.MkdirTemp(dir, "direct-*")
		if err != nil {
			return ctx, dir, err
		}
		probe := newLinkExecutionContext(ctx, plan)
		configureEmscriptenDirectLink(&probe.crossCompile)
		feature := "scheduler"
		if plan.directWasmAllocator {
			configureEmscriptenLibcLink(probe)
			feature = "Go allocator"
		}
		probe.crossCompile.LDFLAGS = append(probe.crossCompile.LDFLAGS, "-sERROR_ON_UNDEFINED_SYMBOLS=1")
		var diagnostics bytes.Buffer
		probe.linkDiagnostics = &diagnostics
		attempt := local
		attempt.outputPath = filepath.Join(probeDir, filepath.Base(output))
		attempt.linkInputs = plan.directWasmInputs
		if err := executeMainLink(probe, &attempt, verbose); err == nil {
			_, _ = diagnostics.WriteTo(os.Stderr)
			if ctx.shouldPrintCommands(verbose) {
				fmt.Fprintf(os.Stderr, "llgo: omit Emscripten %s after link reachability check\n", feature)
			}
			return probe, probeDir, nil
		}
		if err := os.RemoveAll(probeDir); err != nil {
			return ctx, dir, err
		}
		if ctx.shouldPrintCommands(verbose) {
			fmt.Fprintf(os.Stderr, "llgo: retain Emscripten %s after link reachability check\n", feature)
		}
	}
	return ctx, dir, executeMainLink(ctx, &local, verbose)
}
