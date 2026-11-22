package build

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/sizereport"
	"github.com/xgo-dev/llvm"
)

const wasiConcurrentGC = "__llgo_wasi_gc_concurrent"

func canProbeWASIMutator(ctx *context, inputs *linkPruningInputs) bool {
	conf := ctx.buildConf
	if !wasiSizePruningConfig(conf, ctx.crossCompile, ctx.frontendOptions) ||
		conf.ltoMode() != lto.Full || conf.DebugArtifactMode != DebugArtifactNone ||
		conf.PCLNMode == PCLNExternal || conf.GenLL || conf.CheckLLFiles ||
		!ctx.wasmClosedProgram || ctx.wasmRuntimeFree || len(ctx.initial) != 1 ||
		inputs == nil || inputs.opaqueAsmPkg != "" || len(conf.SourceOverlay) != 0 ||
		conf.LinkOptions.ExternalLinkerFlags != "" || os.Getenv("LDFLAGS") != "" || ctx.commands.lookup("LDFLAGS") != "" {
		return false
	}
	for _, pkg := range ctx.pkgByID {
		if !wasmSchedulerProbePackage(pkg.Package, inputs.goroot) ||
			pkg.AltPkg != nil && !wasmSchedulerProbePackage(pkg.AltPkg.Package, inputs.goroot) {
			return false
		}
	}
	return true
}

func buildWASIMutatorSelection(ctx *context, exportFile string) (string, error) {
	pkg := ctx.prog.NewPackage("", "wasi.mutator")
	defer pkg.Module().Dispose()
	guard := pkg.NewVarEx(wasiConcurrentGC, ctx.prog.Pointer(ctx.prog.Byte()))
	guard.Init(ctx.prog.IntVal(0, ctx.prog.Byte()))
	global := pkg.Module().NamedGlobal(wasiConcurrentGC)
	global.SetGlobalConstant(true)
	global.SetVisibility(llvm.HiddenVisibility)
	return exportObject(ctx, "wasi_mutator", exportFile+"-wasi-mutator", pkg)
}

// Each supported WASI worker must be started via a thread-spawn import.
// Keep the ordinary Go-thread runtime for opaque imports and exports too.
func singleWASIMutator(path string) bool {
	report, err := sizereport.Collect(path, nil, "")
	if err != nil || report.Wasm == nil || len(report.Wasm.Memories) != 1 {
		return false
	}
	memory := report.Wasm.Memories[0]
	if !memory.Imported || !memory.Shared || memory.Memory64 {
		return false
	}
	for _, imp := range report.Wasm.Imports {
		if imp.Kind == "memory" && imp.Module == "env" && imp.Name == "memory" {
			continue
		}
		if imp.Kind != "function" || imp.Module != "wasi_snapshot_preview1" ||
			imp.Name == "thread-spawn" || imp.Name == "thread_spawn" {
			return false
		}
	}
	hasStart := false
	for _, exp := range report.Wasm.Exports {
		switch exp.Name {
		case "memory":
			if exp.Kind != "memory" {
				return false
			}
		case "_start", "malloc", "free", "wasi_thread_start":
			if exp.Kind != "function" {
				return false
			}
			hasStart = hasStart || exp.Name == "_start"
		default:
			return false
		}
	}
	return hasStart
}

func probeWASIMutatorLink(ctx *context, plan *mainLinkPlan, dir string, verbose bool) (*context, string, error) {
	probeDir, err := os.MkdirTemp(dir, "wasi-mutator-*")
	if err != nil {
		return nil, "", err
	}
	probe := newLinkExecutionContext(ctx, plan)
	attempt := *plan
	attempt.outputPath = filepath.Join(probeDir, filepath.Base(plan.outputPath))
	attempt.linkInputs = slices.Concat(plan.linkInputs[:1], []string{plan.wasiMutatorSelection}, plan.linkInputs[1:])
	var diagnostics bytes.Buffer
	probe.linkDiagnostics = &diagnostics
	if err := executeMainLink(probe, &attempt, verbose); err == nil && singleWASIMutator(attempt.outputPath) {
		_, _ = diagnostics.WriteTo(os.Stderr)
		if ctx.shouldPrintCommands(verbose) {
			fmt.Fprintln(os.Stderr, "llgo: link WASI collector for one mutator")
		}
		return probe, probeDir, nil
	}
	if ctx.shouldPrintCommands(verbose) {
		fmt.Fprintln(os.Stderr, "llgo: retain WASI concurrent collector")
	}
	return nil, "", os.RemoveAll(probeDir)
}
