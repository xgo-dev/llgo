package build

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
)

const (
	embeddedFullStdout    = "os.llgoFullStdout"
	embeddedStartupTag    = "llgo.embedded.startup"
	embeddedStartupMarker = "__llgo_embedded_requires_runtime_startup"
)

func defineEmbeddedStartupMarker(pkg llssa.Package) {
	fn := pkg.NewFunc(embeddedStartupMarker, llssa.NoArgsNoRet, llssa.CDecl)
	pkg.Module().NamedFunction(embeddedStartupMarker).SetVisibility(llvm.HiddenVisibility)
	fn.MakeBody(1).Return()
}

// Both the frontend and the link use this package-local guard set. Only the
// entry module selects which guards to disable.
var embeddedStartupPackages = []string{"errors", "io", "internal/oserror", "syscall", "internal/poll", "strconv", "time", llssa.PkgRuntime}

func configureEmbeddedStartup(ctx *context, goroot string, inputs *linkPruningInputs) {
	conf, stdout := ctx.buildConf, ctx.stdout
	// The descriptor selector has already rejected opaque native entry
	// points and raw symbol aliases. Compact startup also needs unobserved
	// write results and an archive-independent runtime dependency check.
	if !ctx.embeddedClosedProgram || inputs == nil || len(inputs.roots) != 0 ||
		stdout == nil || len(stdout.builtinCalls) != 0 || stdout.selected == nil || stdout.selected != stdout.descriptor || stdout.discard == nil || conf.Mode != ModeBuild && conf.Mode != ModeRun ||
		conf.Tags != "" || !embeddedSizePruningConfig(conf, ctx.crossCompile, ctx.frontendOptions) {
		return
	}

	if conf.PCLNMode == PCLNExternal || conf.PCLNLoad.startsAtEntry() {
		return
	}
	for _, key := range []string{"LDFLAGS", "CCFLAGS", "CFLAGS", "CGO_CFLAGS", "CGO_LDFLAGS"} {
		if os.Getenv(key) != "" || ctx.commands.lookup(key) != "" {
			return
		}
	}
	// Treat explicit blank imports as requests to keep initialization.
	for _, pkg := range ctx.pkgByID {
		if isStandardLibraryPackage(pkg.Package, goroot) || isRuntimePkg(pkg.PkgPath) {
			continue
		}
		for _, file := range pkg.Syntax {
			for _, imp := range file.Imports {
				if imp.Name != nil && imp.Name.Name == "_" {
					path, _ := strconv.Unquote(imp.Path.Value)
					if slices.Contains(embeddedStartupPackages, path) {
						return
					}
				}
			}
		}
	}
	p := newStartupEffects(ctx, goroot)
	discardCalls := make(map[*ssa.Call]bool, len(stdout.calls))
	for _, instruction := range stdout.calls {
		call, ok := instruction.(*ssa.Call)
		if !ok || call.Referrers() == nil {
			return
		}
		for _, ref := range *call.Referrers() {
			if _, debug := ref.(*ssa.DebugRef); !debug {
				return
			}
		}
		discardCalls[call] = true
	}
	linkname := func(fn *ssa.Function) string {
		if fn.Pkg != nil {
			name, _ := ctx.prog.Linkname(llssa.FullName(fn.Pkg.Pkg, fn.Name()))
			return name
		}
		return ""
	}
	globalSafe := func(global *ssa.Global) bool {
		if p.trusted[global.Pkg.Pkg] {
			return false
		}
		name := llssa.FullName(global.Pkg.Pkg, global.Name())
		if _, linked := ctx.prog.Linkname(name); linked || ctx.prog.HasLinknameTarget(name) {
			return false
		}
		_, locality, local, err := ctx.prog.ResolveLocalityFor(global.Pkg.Pkg, name)
		return err == nil && (!local || locality.Locality == llssa.LocalityNone)
	}
	allowCall := func(call *ssa.Call) bool {
		callee := call.Common().StaticCallee()
		return discardCalls[call] || callee != nil && callee.Pkg != nil && p.trusted[callee.Pkg.Pkg] &&
			callee.Synthetic == "package initializer" && call.Parent().Synthetic == "package initializer"
	}
	ok, usesC := runtimeFreeFunctions(wasmReflectRoots(ctx), linkname, globalSafe, allowCall, false)
	if !ok || usesC {
		return
	}

	// Prove the standard-library startup bodies before changing the entry.
	// The core runtime declares its startup requirements at their use sites.
	dropped := make(map[*ssa.Package]bool)
	for _, path := range embeddedStartupPackages {
		if path == llssa.PkgRuntime {
			continue
		}
		pkg := ctx.progSSA.ImportedPackage(path)
		if pkg != nil {
			dropped[pkg] = true
			if err := p.check(pkg.Func("init")); err != nil {
				if ctx.shouldPrintCommands(conf.Verbose) {
					fmt.Fprintf(os.Stderr, "llgo: retain firmware initialization (%s)\n", err)
				}
				return
			}
		}
	}
	if !checkStartupReads(ctx, &p, dropped) {
		return
	}
	for call := range discardCalls {
		call.Common().Value = stdout.discard
	}
	ctx.probeEmbeddedStartup = true
}

func buildCompactEmbeddedEntry(ctx *context, pkg *packages.Package, preparation *mainLinkPreparation, verbose bool) (string, error) {
	gen := preparation.gen
	gen.rtInit = false
	gen.packageInits = slices.DeleteFunc(slices.Clone(gen.packageInits), func(name string) bool {
		path, ok := strings.CutSuffix(name, ".init")
		return ok && slices.Contains(embeddedStartupPackages, path)
	})
	previousConsole := ctx.embeddedDirectConsole
	ctx.embeddedDirectConsole = true
	entry := genMainModule(ctx, llssa.PkgRuntime, pkg, &gen)
	ctx.embeddedDirectConsole = previousConsole
	cl.DefineInitializerGuard(entry.LPkg, embeddedFullStdout, false)
	for _, guard := range ctx.initializerGuardsOff {
		cl.DefineInitializerGuard(entry.LPkg, guard, false)
	}
	for _, path := range embeddedStartupPackages {
		cl.DefineInitializerGuard(entry.LPkg, cl.PackageInitGuardName(path), false)
	}
	if ctx.buildConf.deadcodeDropEnabled() {
		roots := appendModuleDeclarationRoots(slices.Clone(ctx.sizePruningRoots), entry.LPkg.Module())
		if err := applyDeadcodeDropOverrides(preparation.linkedOrder, entry, false, true, verbose, roots...); err != nil {
			return "", err
		}
	}
	return exportObject(ctx, "entry_compact_main", entry.ExportFile+"-compact", entry.LPkg)
}

func linkCompactEmbeddedEntry(ctx *context, plan *mainLinkPlan, output string, verbose bool) (bool, error) {
	omitFloat := plan.embeddedPrintfFloatMarker != ""
	inputs := slices.Concat([]string{plan.embeddedStartupEntry}, plan.linkInputs[1:])
	linked, err := tryEmbeddedLink(ctx, plan, output, inputs, omitFloat, verbose, embeddedStartupMarker, espPrintfFloatMarker)
	if err == nil && ctx.shouldPrintCommands(verbose) {
		if linked {
			fmt.Fprintln(os.Stderr, "llgo: select firmware console and passive startup")
			if omitFloat {
				fmt.Fprintln(os.Stderr, "llgo: omit unused ESP32-C3 floating-point printf")
			}
		} else {
			fmt.Fprintln(os.Stderr, "llgo: retain firmware initialization (live runtime dependency)")
		}
	}
	return linked, err
}
