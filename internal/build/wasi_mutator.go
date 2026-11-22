package build

import (
	"bytes"
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"slices"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/sizereport"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
)

const (
	wasiConcurrentGC        = "__llgo_wasi_gc_concurrent"
	wasiMutatorSelectionTag = "llgo.wasi.mutatorselect"
	wasiSizeAllocatorLimit  = "-Wl,--max-memory=536870912"
)

func wasiMutatorProfile(conf *Config, target crosscompile.Export, gc bool) bool {
	return gc && wasiSizePruningConfig(conf, target, cl.Options{}) &&
		conf.ltoMode() == lto.Full && conf.DebugArtifactMode == DebugArtifactNone &&
		conf.PCLNMode != PCLNExternal && !conf.GenLL && !conf.CheckLLFiles
}

func canProbeWASIMutator(ctx *context, inputs *linkPruningInputs) bool {
	conf := ctx.buildConf
	if !slices.Contains(ctx.crossCompile.BuildTags, wasiMutatorSelectionTag) ||
		!wasiSizePruningConfig(conf, ctx.crossCompile, ctx.frontendOptions) ||
		!ctx.wasmClosedProgram || ctx.wasmRuntimeFree || len(ctx.initial) != 1 ||
		inputs == nil || inputs.opaqueAsmPkg != "" || len(conf.SourceOverlay) != 0 ||
		conf.LinkOptions.ExternalLinkerFlags != "" || os.Getenv("LDFLAGS") != "" || ctx.commands.lookup("LDFLAGS") != "" {
		return false
	}
	application := make(map[*types.Package]bool)
	for _, pkg := range ctx.pkgByID {
		if !wasmSchedulerProbePackage(pkg.Package, inputs.goroot) ||
			pkg.AltPkg != nil && !wasmSchedulerProbePackage(pkg.AltPkg.Package, inputs.goroot) {
			return false
		}
		if !isStandardLibraryPackage(pkg.Package, inputs.goroot) && !isSchedulerRuntimePackage(pkg.Package) {
			application[pkg.Types] = true
		}
	}
	// Treat reachable go statements in application code as sufficient to
	// retain the concurrent collector. The Go call graph can include library
	// paths that LTO discards, so check library thread creation after linking.
	for fn := range wasmProgramUseFor(ctx).reachable {
		if fn.Pkg == nil || !application[fn.Pkg.Pkg] {
			continue
		}
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				if _, startsGoroutine := instr.(*ssa.Go); startsGoroutine {
					if ctx.shouldPrintCommands(conf.Verbose) {
						fmt.Fprintln(os.Stderr, "llgo: retain WASI concurrent collector (reachable go statement)")
					}
					return false
				}
			}
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

func probeWASIMutatorLink(ctx *context, plan *mainLinkPlan, dir string, verbose bool, compactEntry string) (*context, string, error) {
	probeDir, err := os.MkdirTemp(dir, "wasi-mutator-*")
	if err != nil {
		return nil, "", err
	}
	probe := newLinkExecutionContext(ctx, plan)
	attempt := *plan
	attempt.outputPath = filepath.Join(probeDir, filepath.Base(plan.outputPath))
	entry := plan.linkInputs[0]
	if compactEntry != "" {
		entry = compactEntry
		attempt.embeddedPCLN, attempt.entryIR = nil, ""
	}
	attempt.linkInputs = slices.Concat([]string{entry, plan.wasiMutatorSelection}, plan.linkInputs[1:])
	var diagnostics bytes.Buffer
	probe.linkDiagnostics = &diagnostics
	allocator := filepath.Join(probeDir, "emmalloc.o")
	defer os.Remove(allocator)
	smallAllocator := slices.Contains(probe.crossCompile.LDFLAGS, wasiSizeAllocatorLimit) &&
		compileWASISizeAllocator(probe, allocator) == nil
	if smallAllocator {
		// wasm-ld extracts the libc allocator when it encounters -lc.
		probe.crossCompile.LDFLAGS = slices.Concat([]string{allocator}, probe.crossCompile.LDFLAGS)
	} else {
		diagnostics.Reset()
	}
	err = executeMainLink(probe, &attempt, verbose)
	if err != nil && smallAllocator {
		probe = newLinkExecutionContext(ctx, plan)
		diagnostics.Reset()
		probe.linkDiagnostics = &diagnostics
		smallAllocator = false
		err = executeMainLink(probe, &attempt, verbose)
	}
	if err == nil && singleWASIMutator(attempt.outputPath) {
		_, _ = diagnostics.WriteTo(os.Stderr)
		if ctx.shouldPrintCommands(verbose) {
			if compactEntry != "" {
				fmt.Fprintln(os.Stderr, "llgo: select compact WASI startup")
			}
			fmt.Fprintln(os.Stderr, "llgo: link WASI collector for one mutator")
			if smallAllocator {
				fmt.Fprintln(os.Stderr, "llgo: use compact WASI libc allocator")
			}
		}
		return probe, probeDir, nil
	}
	if ctx.shouldPrintCommands(verbose) {
		if compactEntry != "" {
			fmt.Fprintln(os.Stderr, "llgo: fall back to ordinary WASI startup")
		} else {
			fmt.Fprintln(os.Stderr, "llgo: retain WASI concurrent collector")
		}
	}
	return nil, "", os.RemoveAll(probeDir)
}

// Keep the upstream allocator's region aliasing behavior under Clang
// optimization. A separate object lets the executable select its allocator
// without changing the runtime or application archives.
func compileWASISizeAllocator(ctx *context, output string) error {
	source := filepath.Join(env.LLGoRuntimeDir(), "internal", "runtime", "tinygogc", "_wasi_allocator", "emmalloc.c")
	cmd := ctx.compiler()
	cmd.Stderr = ctx.linkDiagnostics
	return cmd.CompileWithoutLTO("-x", "c", "-Oz", "-g0", "-DNDEBUG",
		"-fno-strict-aliasing", "-o", output, "-c", source)
}
