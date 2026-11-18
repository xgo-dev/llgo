package build

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"os"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

type mapFormatPlan struct {
	callee *ssa.Function
	helper *ssa.Function
	calls  map[*ssa.CallCommon]*types.Package
}

var stringIntMap = types.NewMap(types.Typ[types.String], types.Typ[types.Int])

func (ctx *context) newMapFormatPlan() *mapFormatPlan {
	if ctx.frontend != nil || ctx.buildConf.BuildMode != BuildModeExe ||
		ctx.crossCompile.WasmProfile != crosscompile.WasmProfileW32 ||
		ctx.crossCompile.WasmProvider != crosscompile.WasmProviderWASI {
		return nil
	}
	pkg := ctx.progSSA.ImportedPackage("fmt")
	if pkg == nil {
		return nil
	}
	callee, helper := pkg.Func("Sprintf"), pkg.Func("llgoSprintfStringIntMap")
	if callee == nil || helper == nil {
		return nil
	}
	return &mapFormatPlan{callee: callee, helper: helper}
}

func (plan *mapFormatPlan) record(instruction ssa.Instruction, owner *ssa.Package) {
	call, ok := instruction.(*ssa.Call)
	if !ok || owner == nil || owner == plan.callee.Pkg {
		return
	}
	common := call.Common()
	if common.StaticCallee() != plan.callee || len(common.Args) != 2 {
		return
	}
	format, ok := common.Args[0].(*ssa.Const)
	if !ok || format.Value == nil || format.Value.Kind() != constant.String || constant.StringVal(format.Value) != "%v" {
		return
	}
	operands := formatOperands(call, common.Args[1])
	if len(operands) != 1 || operands[0] == nil || !types.Identical(operands[0], stringIntMap) {
		return
	}
	if plan.calls == nil {
		plan.calls = make(map[*ssa.CallCommon]*types.Package)
	}
	plan.calls[common] = owner.Pkg
}

// Only replace the calls if doing so can release the general formatter. Use
// the same input proof as release DCE, including its translated assembly roots.
func configureMapFormats(ctx *context, in *wasiSizePruningInputs) {
	plan := ctx.mapFormatPlan
	ctx.mapFormatPlan = nil
	if plan == nil || len(plan.calls) == 0 {
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
		roots, opaque = mapFormatExportRoots(ctx.progSSA, roots, pkg.Package, in.goroot)
		if !opaque && pkg.AltPkg != nil {
			roots, opaque = mapFormatExportRoots(ctx.progSSA, roots, pkg.AltPkg.Package, in.goroot)
		}
		if opaque {
			if verbose {
				fmt.Fprintf(os.Stderr, "llgo: retain generic fmt: foreign entry in %s\n", pkg.PkgPath)
			}
			return
		}
	}
	asmRoots := make(map[string]bool, len(in.roots))
	for _, name := range in.roots {
		asmRoots[name] = true
	}
	for _, pkg := range ctx.progSSA.AllPackages() {
		for name, member := range pkg.Members {
			if fn, ok := member.(*ssa.Function); ok && (asmRoots[llssa.FullName(pkg.Pkg, name)] ||
				pkg.Pkg.Path() == llssa.PkgRuntime && name == "init") {
				roots = append(roots, fn)
			}
		}
	}
	for call := range plan.calls {
		call.Value = plan.helper
	}
	for fn := range analyzeWasmProgramUse(ctx.progSSA, roots).reachable {
		if fn.Pkg == plan.callee.Pkg && (fn.Name() == "printArg" || fn.Name() == "printValue") {
			for call := range plan.calls {
				call.Value = plan.callee
			}
			if verbose {
				fmt.Fprintln(os.Stderr, "llgo: retain generic fmt: general formatting is reachable")
			}
			return
		}
	}
	ctx.mapFormatPackages = make(map[*types.Package]bool)
	for _, owner := range plan.calls {
		ctx.mapFormatPackages[owner] = true
	}
	if verbose {
		fmt.Fprintln(os.Stderr, "llgo: specialize map formatting")
	}
}

// A modeled //export declaration is an extra RTA root. Other export
// directives and aliases in user code can enter Go outside the SSA call graph.
func mapFormatExportRoots(prog *ssa.Program, roots []*ssa.Function, pkg *packages.Package, goroot string) ([]*ssa.Function, bool) {
	if pkg == nil {
		return roots, false
	}
	standard := isStandardLibraryPackage(pkg, goroot) || pkg.Module != nil && pkg.Module.Path == env.LLGoRuntimePkg
	ssaPkg := prog.Package(pkg.Types)
	for _, file := range pkg.Syntax {
		for _, group := range file.Comments {
			for _, item := range directive.ParseGroup(group) {
				switch item.Name {
				case "go:wasmexport", "llgo:export":
					return roots, true
				case "go:linkname", "llgo:link":
					if !standard {
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
