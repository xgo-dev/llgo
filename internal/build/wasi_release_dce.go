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
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	"github.com/xgo-dev/llvm"
)

// linkPruningInputs tracks assembly references and foreign link inputs that
// do not appear in the Go call graph.
type linkPruningInputs struct {
	goroot       string
	roots        []string
	opaqueAsmPkg string
	nativeFormat bool
}

func (ctx *context) wasiSizePruningInputs(goroot string) *linkPruningInputs {
	conf := ctx.buildConf
	if (conf.deadcodeDropEnabled() || conf.goGlobalDCEEnabled()) && ctx.valueFormatPlan == nil {
		return nil
	}
	if !wasiSizePruningConfig(conf, ctx.crossCompile, ctx.frontendOptions) || ctx.wasmRuntimeFree || ctx.frontend != nil {
		return nil
	}
	// Full LTO consumes the strong overrides in the merged module. ThinLTO
	// resolves archive ownership separately, so release builds use its
	// conservative link until that combination has been validated.
	if conf.ltoMode() == lto.Thin {
		if conf.Verbose || conf.PrintCommands {
			fmt.Fprintln(os.Stderr, "llgo: retain WASI code and metadata: ThinLTO")
		}
		return nil
	}
	if conf.LTOPlugin.Enabled() {
		if conf.Verbose || conf.PrintCommands {
			fmt.Fprintln(os.Stderr, "llgo: retain WASI code and metadata: LTO pass plugin")
		}
		return nil
	}
	if len(wasmReflectRoots(ctx)) == 0 {
		return nil
	}
	return &linkPruningInputs{goroot: goroot}
}

func wasiSizePruningConfig(conf *Config, target crosscompile.Export, frontend cl.Options) bool {
	return (conf.Mode == ModeBuild || conf.Mode == ModeRun || conf.Mode == ModeTest) &&
		conf.BuildMode == BuildModeExe && conf.OptLevel == optlevel.Oz && conf.WasmOpt == WasmOptSize &&
		(conf.Target == "" || conf.Target == "wasi" || conf.Target == "wasip1") &&
		target.WasmProfile == crosscompile.WasmProfileW32 && target.WasmProvider == crosscompile.WasmProviderWASI &&
		len(target.ExtraFiles) == 0 && conf.LinkOptions.ExternalLinker == "" &&
		conf.Coverage == nil && conf.ModuleHook == nil && !frontend.Debug && !frontend.DebuggerFrames
}

// Alternate and bundled native packages bypass the initializer pass's normal
// assembly inspection. A selected assembly body here precludes size pruning.
func (in *linkPruningInputs) checkAlternateAsm(ctx *context, pkg *packages.Package) {
	files, err := compiledPkgSFiles(ctx, pkg)
	if err != nil {
		in.opaqueAsmPkg = pkg.PkgPath
		return
	}
	for _, path := range files {
		mod, err := translatePruningAssembly(ctx, pkg, path)
		if err != nil || !mod.IsNil() {
			if !mod.IsNil() {
				mod.Dispose()
			}
			in.opaqueAsmPkg = pkg.PkgPath
			return
		}
	}
}

func (in *linkPruningInputs) collectAsmReferences(pkg *packages.Package, path string, conf *Config, mod llvm.Module) {
	// Even with complete symbol references, arbitrary assembly could operate
	// directly on Go interfaces or method tables.
	if _, overlaid := conf.Overlay[path]; overlaid || !isStandardLibraryPackage(pkg, in.goroot) {
		in.opaqueAsmPkg = pkg.PkgPath
		return
	}
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				if inst.IsACallInst().IsNil() && inst.IsAInvokeInst().IsNil() {
					continue
				}
				callee := inst.CalledValue()
				if callee.IsAFunction().IsNil() && !(in.nativeFormat && isSystemRegisterAsm(callee)) {
					in.opaqueAsmPkg = pkg.PkgPath
					return
				}
			}
		}
	}
	in.roots = appendModuleDeclarationRoots(in.roots, mod)
}

// The ARM64 translator emits system register access as LLVM inline assembly.
// A single MRS or MSR instruction cannot transfer control to a Go symbol.
func isSystemRegisterAsm(callee llvm.Value) bool {
	if callee.IsAInlineAsm().IsNil() {
		return false
	}
	fields := strings.Fields(callee.InlineAsmInfo().Assembly)
	if len(fields) != 3 {
		return false
	}
	var reg string
	switch strings.ToLower(fields[0]) {
	case "mrs":
		if fields[1] != "$0," {
			return false
		}
		reg = fields[2]
	case "msr":
		if fields[2] != "$0" || !strings.HasSuffix(fields[1], ",") {
			return false
		}
		reg = strings.TrimSuffix(fields[1], ",")
	default:
		return false
	}
	return reg != "" && strings.IndexFunc(reg, func(r rune) bool {
		return r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9')
	}) < 0
}

func appendModuleDeclarationRoots(roots []string, mod llvm.Module) []string {
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if fn.IsDeclaration() && fn.IntrinsicID() == 0 && !fn.FirstUse().IsNil() {
			roots = append(roots, fn.Name())
		}
	}
	for glob := mod.FirstGlobal(); !glob.IsNil(); glob = llvm.NextGlobal(glob) {
		if glob.IsDeclaration() && !glob.FirstUse().IsNil() {
			roots = append(roots, glob.Name())
		}
	}
	return roots
}

// //export is modeled by dceEntryRootCandidates. Future export directives
// must be modeled as roots before they participate in the release path.
func hasUntrackedWasmExport(syntax []*ast.File) bool {
	for _, file := range syntax {
		for _, group := range file.Comments {
			for _, item := range directive.ParseGroup(group) {
				if item.Name == "go:wasmexport" || item.Name == "llgo:export" {
					return true
				}
			}
		}
	}
	return false
}

func configureWasiReleaseDCE(ctx *context, in *linkPruningInputs) {
	if in == nil || ctx.wasmRuntimeFree || !canPruneWasmTypeMetadata(ctx) {
		return
	}
	conf := ctx.buildConf
	if conf.deadcodeDropEnabled() || conf.goGlobalDCEEnabled() {
		return
	}
	if in.opaqueAsmPkg != "" {
		if conf.Verbose || conf.PrintCommands {
			fmt.Fprintf(os.Stderr, "llgo: retain WASI code and metadata: opaque assembly in %s\n", in.opaqueAsmPkg)
		}
		return
	}
	for _, pkg := range ctx.pkgByID {
		if hasUntrackedWasmExport(pkg.Syntax) || pkg.AltPkg != nil && hasUntrackedWasmExport(pkg.AltPkg.Syntax) {
			if conf.Verbose || conf.PrintCommands {
				fmt.Fprintf(os.Stderr, "llgo: retain WASI code and metadata: untracked host export in %s\n", pkg.PkgPath)
			}
			return
		}
	}
	slices.Sort(in.roots)
	ctx.wasiSizeRoots = slices.Compact(in.roots)
	conf.wasiReleaseDCE = true
	ctx.prog.EnableDeadcodeDrop(true)
	if conf.Verbose || conf.PrintCommands {
		fmt.Fprintln(os.Stderr, "llgo: enable closed-program WASI size pruning")
	}
}
