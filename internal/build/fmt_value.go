package build

import (
	"fmt"
	"os"
	"slices"
	"strings"

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
	return (wasiSizePruningConfig(conf, target, frontend) || hostedValueFormatSupport(conf, target, frontend) || embeddedSizePruningConfig(conf, target, frontend)) &&
		conf.ltoMode() != lto.Thin && !conf.LTOPlugin.Enabled() && !frontendCacheEnabled(conf)
}

// Hosted targets use the same typed functions. Require the size-link flags
// under which removing the general formatter has been validated.
func hostedValueFormatSupport(conf *Config, target crosscompile.Export, frontend cl.Options) bool {
	if (conf.Mode != ModeBuild && conf.Mode != ModeRun && conf.Mode != ModeTest) ||
		conf.OptLevel != optlevel.Oz || conf.BuildMode != BuildModeExe ||
		conf.ltoMode() != lto.Full || !conf.deadcodeDropEnabled() || conf.goGlobalDCEEnabled() ||
		conf.LinkOptions.ExternalLinker != "" || len(target.ExtraFiles) != 0 ||
		conf.Coverage != nil || conf.ModuleHook != nil || frontend.Debug || frontend.DebuggerFrames {
		return false
	}
	if conf.Goarch == "wasm" {
		return target.WasmProvider == crosscompile.WasmProviderEmscripten &&
			(target.WasmProfile == crosscompile.WasmProfileJ32 || target.WasmProfile == crosscompile.WasmProfileJ64) &&
			conf.WasmOpt == WasmOptSize
	}
	return conf.Target == "" && conf.Goos == "darwin" && conf.Goarch == "arm64"
}

// Resolve typed rewrites before init and caller accounting. For hosted targets
// use the same checks for unknown native code and assembly as WASI.
func configureFormattingAndInits(ctx *context, goroot string, inputs *linkPruningInputs) *linkPruningInputs {
	formatInputs := inputs
	if formatInputs == nil &&
		(hostedValueFormatSupport(ctx.buildConf, ctx.crossCompile, ctx.frontendOptions) ||
			(len(ctx.jsonCodecPackages) != 0 && linuxSizeCodecBuild(ctx.buildConf, ctx.crossCompile, ctx.frontendOptions))) {
		formatInputs = &linkPruningInputs{goroot: goroot, nativeFormat: ctx.buildConf.Goarch != "wasm"}
		if ctx.buildConf.Goarch != "wasm" {
			if opaque := opaqueGlobalInitPackage(ctx, make(map[string]bool), formatInputs); opaque != "" {
				if ctx.buildConf.Verbose || ctx.buildConf.PrintCommands {
					fmt.Fprintf(os.Stderr, "llgo: retain generic fmt: unknown native input in %s\n", opaque)
				}
				formatInputs = nil
			}
		}
	}
	configureUnusedGlobalInits(ctx, formatInputs)
	if embeddedSizePruningConfig(ctx.buildConf, ctx.crossCompile, ctx.frontendOptions) {
		if !ctx.embeddedClosedProgram {
			return nil
		}
	} else {
		configureValueFormats(ctx, formatInputs)
	}
	return formatInputs
}

// Darwin's runtime and syscall packages use these system libraries without
// introducing application-owned native objects.
func allowsNativeFormatImports(ctx *context, in *linkPruningInputs, pkg *packages.Package, flags []string, imports []cgoImportDynamicDecl) bool {
	conf := ctx.buildConf
	if in == nil || !in.nativeFormat || conf.Goos != "darwin" || !isStandardLibraryPackage(pkg, in.goroot) {
		return false
	}
	for _, path := range slices.Concat(pkg.GoFiles, pkg.CompiledGoFiles) {
		if _, replaced := conf.SourceOverlay[path]; replaced {
			return false
		}
	}
	if pkg.PkgPath != "syscall" && pkg.PkgPath != "internal/syscall/unix" && pkg.PkgPath != "runtime" {
		return false
	}
	for _, flag := range flags {
		if flag != "-lresolv" || pkg.PkgPath != "internal/syscall/unix" {
			return false
		}
	}
	for _, imp := range imports {
		switch {
		case (imp.library == "/usr/lib/libSystem.B.dylib" || imp.library == "") && strings.HasPrefix(imp.local, "libc_"):
		case imp.library == "/usr/lib/libresolv.9.dylib" && strings.HasPrefix(imp.local, "libresolv_") &&
			pkg.PkgPath == "internal/syscall/unix":
		default:
			return false
		}
	}
	return true
}

// Only replace the calls if doing so can release the general formatter. Use
// the same input proof as release DCE, including its translated assembly roots.
func configureValueFormats(ctx *context, in *linkPruningInputs) {
	plan := ctx.valueFormatPlan
	ctx.valueFormatPlan = nil
	if !plan.HasCalls() {
		return
	}
	verbose := ctx.buildConf.Verbose || ctx.buildConf.PrintCommands
	if in == nil || ctx.buildConf.Goarch == "wasm" && !ctx.wasmClosedProgram || in.opaqueAsmPkg != "" {
		if verbose {
			fmt.Fprintln(os.Stderr, "llgo: retain generic fmt: open link inputs")
			if in != nil && in.opaqueAsmPkg != "" {
				fmt.Fprintf(os.Stderr, "llgo: opaque assembly for format reachability in %s\n", in.opaqueAsmPkg)
			}
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
