package build

import (
	"fmt"
	"os"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/build/fmtspecialize"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func canSpecializeFormats(ctx *context) bool {
	conf := ctx.buildConf
	return conf != nil && conf.OptLevel != optlevel.O0 &&
		ctx.mode != ModeGen && !ctx.frontendOptions.DebuggerFrames
}

func prepareValueFormatSupport(conf *Config, target crosscompile.Export, frontend cl.Options) bool {
	return wasiSizePruningConfig(conf, target, frontend) &&
		conf.ltoMode() != lto.Thin && !conf.LTOPlugin.Enabled() && !frontendCacheEnabled(conf)
}

// Only replace the calls if doing so can release the general formatter. Use
// the same input proof as release DCE, including its translated assembly roots.
func configureValueFormats(ctx *context, in *wasiSizePruningInputs) {
	plan := ctx.valueFormatPlan
	ctx.valueFormatPlan = nil
	if !plan.HasCalls() {
		return
	}
	verbose := ctx.buildConf.Verbose || ctx.buildConf.PrintCommands
	if in == nil || !canPruneWasmTypeMetadata(ctx) || in.opaqueAsmPkg != "" {
		if verbose {
			fmt.Fprintln(os.Stderr, "llgo: retain generic fmt: open link inputs")
		}
		return
	}
	roots := wasmReflectRoots(ctx)
	for _, pkg := range ctx.pkgByID {
		var opaque bool
		roots, opaque = formatExportRoots(ctx.progSSA, roots, pkg.Package, in.goroot)
		if !opaque && pkg.AltPkg != nil {
			roots, opaque = formatExportRoots(ctx.progSSA, roots, pkg.AltPkg.Package, in.goroot)
		}
		if opaque {
			if verbose {
				fmt.Fprintf(os.Stderr, "llgo: retain generic fmt: foreign entry in %s\n", pkg.PkgPath)
			}
			return
		}
	}
	if pkg := ctx.progSSA.ImportedPackage(llssa.PkgRuntime); pkg != nil {
		if init := pkg.Func("init"); init != nil {
			roots = append(roots, init)
		}
	}
	roots = fmtspecialize.AppendSymbolRoots(ctx.progSSA, roots, in.roots)
	ctx.valueFormatPackages = plan.ApplyStandalone(roots)
	if len(ctx.valueFormatPackages) == 0 {
		if verbose {
			fmt.Fprintln(os.Stderr, "llgo: retain generic fmt: general formatting is reachable")
		}
		return
	}
	if verbose {
		fmt.Fprintln(os.Stderr, "llgo: specialize default value formatting")
	}
}

func formatExportRoots(prog *ssa.Program, roots []*ssa.Function, pkg *packages.Package, goroot string) ([]*ssa.Function, bool) {
	trusted := pkg != nil && (isStandardLibraryPackage(pkg, goroot) || pkg.Module != nil && pkg.Module.Path == env.LLGoRuntimePkg)
	return fmtspecialize.AppendExportRoots(prog, roots, pkg, trusted)
}
