package build

import (
	"fmt"
	"go/ast"
	"os"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func sizeCodecBuild(conf *Config, frontend cl.Options) bool {
	return conf.OptLevel == optlevel.Oz && conf.ltoMode() == lto.Full && conf.deadcodeDropEnabled() &&
		conf.BuildMode == BuildModeExe && conf.Mode != ModeGen && conf.Coverage == nil &&
		conf.ModuleHook == nil && !frontend.Debug && !frontend.DebuggerFrames &&
		!conf.LTOPlugin.Enabled() && !frontendCacheEnabled(conf) && len(conf.SourceOverlay) == 0
}

func linuxSizeCodecBuild(conf *Config, target crosscompile.Export, frontend cl.Options) bool {
	return !conf.disableSizeSpecialization && conf.Target == "" && conf.Goos == "linux" &&
		(conf.Goarch == "amd64" || conf.Goarch == "arm64") &&
		len(target.ExtraFiles) == 0 && conf.LinkOptions.ExternalLinker == "" &&
		conf.LinkOptions.ExternalLinkerFlags == "" && sizeCodecBuild(conf, frontend)
}

// Hosted targets use the same closed-link size checks as WASI. Require the
// size-link flags under which link-time pruning has been validated.
func hostedValueFormatSupport(conf *Config, target crosscompile.Export, frontend cl.Options) bool {
	if conf.disableSizeSpecialization || (conf.Mode != ModeBuild && conf.Mode != ModeRun && conf.Mode != ModeTest) ||
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
	return (conf.Target == "" && conf.Goos == "darwin" && conf.Goarch == "arm64") ||
		linuxSizeCodecBuild(conf, target, frontend)
}

// Resolve closed-link native/embedded/WASM pruning inputs before initializer
// guards and release DCE run.
func configureClosedLinkPruning(ctx *context, goroot string, inputs *linkPruningInputs) *linkPruningInputs {
	formatInputs := inputs
	hosted := hostedValueFormatSupport(ctx.buildConf, ctx.crossCompile, ctx.frontendOptions)
	if formatInputs == nil && hosted {
		formatInputs = &linkPruningInputs{goroot: goroot, nativeFormat: ctx.buildConf.Goarch != "wasm"}
		if ctx.buildConf.Goarch != "wasm" {
			if opaque := opaqueGlobalInitPackage(ctx, make(map[string]bool), formatInputs); opaque != "" {
				if ctx.buildConf.Verbose || ctx.buildConf.PrintCommands {
					fmt.Fprintf(os.Stderr, "llgo: retain generic fmt: unknown native input in %s\n", opaque)
				}
				formatInputs = nil
			}
			if formatInputs != nil && formatInputs.opaqueAsmPkg == "" &&
				ctx.frontend == nil && len(ctx.initial) == 1 && !hasOpaqueFormatExports(ctx, goroot) {
				ctx.nativeClosedProgram = true
				slices.Sort(formatInputs.roots)
				formatInputs.roots = slices.Compact(formatInputs.roots)
				ctx.sizePruningRoots = formatInputs.roots
			}
		}
	}
	configureUnusedGlobalInits(ctx, formatInputs)
	if embeddedSizePruningConfig(ctx.buildConf, ctx.crossCompile, ctx.frontendOptions) && !ctx.embeddedClosedProgram {
		return nil
	}
	return formatInputs
}

func hasOpaqueFormatExports(ctx *context, goroot string) bool {
	for _, pkg := range ctx.pkgByID {
		if _, opaque := formatExportRoots(ctx.progSSA, nil, pkg.Package, goroot); opaque {
			return true
		}
		if pkg.AltPkg != nil {
			if _, opaque := formatExportRoots(ctx.progSSA, nil, pkg.AltPkg.Package, goroot); opaque {
				return true
			}
		}
	}
	return false
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

func formatExportRoots(prog *ssa.Program, roots []*ssa.Function, pkg *packages.Package, goroot string) ([]*ssa.Function, bool) {
	trusted := pkg != nil && (isStandardLibraryPackage(pkg, goroot) || pkg.Module != nil && pkg.Module.Path == env.LLGoRuntimePkg)
	return appendExportRoots(prog, roots, pkg, trusted)
}

// appendSymbolRoots includes Go functions referenced by translated assembly.
func appendSymbolRoots(prog *ssa.Program, roots []*ssa.Function, symbols []string) []*ssa.Function {
	if len(symbols) == 0 {
		return roots
	}
	packagesBySymbolPath := make(map[string][]*ssa.Package)
	for _, pkg := range prog.AllPackages() {
		path := llssa.PathOf(pkg.Pkg)
		packagesBySymbolPath[path] = append(packagesBySymbolPath[path], pkg)
	}
	for _, symbol := range symbols {
		i := strings.LastIndexByte(symbol, '.')
		if i < 0 {
			continue
		}
		for _, pkg := range packagesBySymbolPath[symbol[:i]] {
			if fn := pkg.Func(symbol[i+1:]); fn != nil {
				roots = append(roots, fn)
			}
		}
	}
	return roots
}

// appendExportRoots collects modeled C entry points. The second result
// reports an entry that can enter Go outside the modeled call graph.
func appendExportRoots(prog *ssa.Program, roots []*ssa.Function, pkg *packages.Package, trusted bool) ([]*ssa.Function, bool) {
	if pkg == nil {
		return roots, false
	}
	ssaPkg := prog.Package(pkg.Types)
	for _, file := range pkg.Syntax {
		for _, group := range file.Comments {
			for _, item := range directive.ParseGroup(group) {
				switch item.Name {
				case "go:wasmexport", "llgo:export":
					return roots, true
				case "go:linkname", "llgo:link":
					if !trusted {
						return roots, true
					}
				}
			}
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			for _, item := range directive.ParseGroup(fn.Doc) {
				if item.Name != "export" {
					continue
				}
				if ssaPkg == nil || fn.Recv != nil || ssaPkg.Func(fn.Name.Name) == nil {
					return roots, true
				}
				roots = append(roots, ssaPkg.Func(fn.Name.Name))
			}
		}
	}
	return roots, false
}
