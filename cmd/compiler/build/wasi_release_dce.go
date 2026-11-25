package build

import (
	"fmt"
	"go/ast"
	"os"
	"slices"

	"github.com/xgo-dev/llgo/cmd/compiler/crosscompile"
	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	"github.com/xgo-dev/llgo/directive"
	"github.com/xgo-dev/llvm"
)

// WASI size pruning needs both the initializer pass's native-input proof and
// the function references that don't appear in Go package metadata.
type wasiSizePruningInputs struct {
	goroot       string
	roots        []string
	opaqueAsmPkg string
}

func (ctx *context) wasiSizePruningInputs(goroot string) *wasiSizePruningInputs {
	conf := ctx.buildConf
	if conf.deadcodeDropEnabled() || conf.goGlobalDCEEnabled() || conf.ltoEnabled() ||
		(conf.Mode != ModeBuild && conf.Mode != ModeRun && conf.Mode != ModeTest) ||
		conf.BuildMode != BuildModeExe || conf.OptLevel != optlevel.Oz || conf.WasmOpt != WasmOptSize ||
		(conf.Target != "" && conf.Target != "wasi" && conf.Target != "wasip1") ||
		ctx.crossCompile.WasmProfile != crosscompile.WasmProfileW32 ||
		ctx.crossCompile.WasmProvider != crosscompile.WasmProviderWASI || len(ctx.crossCompile.ExtraFiles) != 0 ||
		conf.LinkOptions.ExternalLinker != "" || ctx.commands.lookup("LDFLAGS") != "" ||
		ctx.wasmRuntimeFree || ctx.frontend != nil ||
		conf.Coverage != nil || conf.ModuleHook != nil || ctx.frontendOptions.Debug || ctx.frontendOptions.DebuggerFrames {
		return nil
	}
	if len(wasmReflectRoots(ctx)) == 0 {
		return nil
	}
	return &wasiSizePruningInputs{goroot: goroot}
}

// Alternate and bundled native packages bypass the initializer pass's normal
// assembly inspection. A selected assembly body here precludes size pruning.
func (in *wasiSizePruningInputs) checkAlternateAsm(ctx *context, pkg *packages.Package) {
	if files, err := compiledPkgSFiles(ctx, pkg); err != nil || len(files) != 0 {
		in.opaqueAsmPkg = pkg.PkgPath
	}
}

func (in *wasiSizePruningInputs) collectAsmReferences(pkg *packages.Package, path string, conf *Config, mod llvm.Module) {
	// Even with complete symbol references, arbitrary assembly could operate
	// directly on Go interfaces or method tables.
	if _, overlaid := conf.Overlay[path]; overlaid || !isStandardLibraryPackage(pkg, in.goroot) {
		in.opaqueAsmPkg = pkg.PkgPath
		return
	}
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				if (!inst.IsACallInst().IsNil() || !inst.IsAInvokeInst().IsNil()) && inst.CalledValue().IsAFunction().IsNil() {
					in.opaqueAsmPkg = pkg.PkgPath
					return
				}
			}
		}
	}
	in.roots = appendModuleDeclarationRoots(in.roots, mod)
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

func configureWasiReleaseDCE(ctx *context, in *wasiSizePruningInputs) {
	if in == nil || !canPruneWasmTypeMetadata(ctx) {
		return
	}
	conf := ctx.buildConf
	if in.opaqueAsmPkg != "" {
		if conf.Verbose || conf.PrintCommands {
			fmt.Fprintf(os.Stderr, "llgo: retain WASI code and metadata: opaque assembly in %s\n", in.opaqueAsmPkg)
		}
		return
	}
	for _, pkg := range ctx.pkgByID {
		if hasUntrackedWasmExport(pkg.Syntax) || pkg.AltPkg != nil && hasUntrackedWasmExport(pkg.AltPkg.Syntax) {
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
