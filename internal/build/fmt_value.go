package build

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"os"
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

type valueFormatPlan struct {
	callee *ssa.Function
	value  *ssa.Function
	maps   *ssa.Function
	calls  map[*ssa.CallCommon]formatRewrite
}

type formatRewrite struct {
	owner    *types.Package
	function *ssa.Function
}

func (ctx *context) newValueFormatPlan() *valueFormatPlan {
	if ctx.frontend != nil || ctx.buildConf.BuildMode != BuildModeExe ||
		ctx.crossCompile.WasmProfile != crosscompile.WasmProfileW32 ||
		ctx.crossCompile.WasmProvider != crosscompile.WasmProviderWASI {
		return nil
	}
	pkg := ctx.progSSA.ImportedPackage("fmt")
	if pkg == nil {
		return nil
	}
	callee := pkg.Func("Sprintf")
	value, maps := pkg.Func("llgoSprintfValue"), pkg.Func("llgoSprintfValueWithMaps")
	if callee == nil || value == nil || maps == nil {
		return nil
	}
	return &valueFormatPlan{callee: callee, value: value, maps: maps}
}

func (plan *valueFormatPlan) record(instruction ssa.Instruction, owner *ssa.Package) {
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
	if len(operands) != 1 || operands[0] == nil {
		return
	}
	hasMap, ok := defaultFormatType(operands[0], true, make(map[types.Type]bool))
	if !ok {
		return
	}
	helper := plan.value
	if hasMap {
		helper = plan.maps
	}
	if plan.calls == nil {
		plan.calls = make(map[*ssa.CallCommon]formatRewrite)
	}
	plan.calls[common] = formatRewrite{owner: owner.Pkg, function: helper}
}

// defaultFormatType checks the default formatter's actual traversal, including
// named types whose methods could replace formatting.
func defaultFormatType(t types.Type, top bool, visiting map[types.Type]bool) (bool, bool) {
	t = types.Unalias(t)
	if visiting[t] {
		return false, false
	}
	visiting[t] = true
	defer delete(visiting, t)
	methods := types.NewMethodSet(t)
	for i := 0; i < methods.Len(); i++ {
		switch methods.At(i).Obj().Name() {
		case "Format", "String", "Error":
			return false, false
		}
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		return false, u.Kind() != types.Invalid && u.Kind() != types.UntypedNil
	case *types.Array:
		return defaultFormatType(u.Elem(), false, visiting)
	case *types.Slice:
		return defaultFormatType(u.Elem(), false, visiting)
	case *types.Struct:
		var hasMap bool
		for i := 0; i < u.NumFields(); i++ {
			nestedMap, ok := defaultFormatType(u.Field(i).Type(), false, visiting)
			if !ok {
				return false, false
			}
			hasMap = hasMap || nestedMap
		}
		return hasMap, true
	case *types.Map:
		_, keyOK := defaultFormatType(u.Key(), false, visiting)
		_, elemOK := defaultFormatType(u.Elem(), false, visiting)
		return true, keyOK && elemOK
	case *types.Pointer:
		if top {
			switch u.Elem().Underlying().(type) {
			case *types.Array, *types.Slice, *types.Struct, *types.Map:
				return defaultFormatType(u.Elem(), false, visiting)
			}
		}
		return false, true
	case *types.Chan, *types.Signature:
		return false, true
	default:
		return false, false
	}
}

// Only replace the calls if doing so can release the general formatter. Use
// the same input proof as release DCE, including its translated assembly roots.
func configureValueFormats(ctx *context, in *wasiSizePruningInputs) {
	plan := ctx.valueFormatPlan
	ctx.valueFormatPlan = nil
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
	if len(in.roots) != 0 {
		packagesBySymbolPath := make(map[string][]*ssa.Package)
		for _, pkg := range ctx.progSSA.AllPackages() {
			path := llssa.PathOf(pkg.Pkg)
			packagesBySymbolPath[path] = append(packagesBySymbolPath[path], pkg)
		}
		for _, symbol := range in.roots {
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
	}
	for call, rewrite := range plan.calls {
		call.Value = rewrite.function
	}
	use := analyzeWasmProgramUse(ctx.progSSA, roots)
	for fn := range use.reachable {
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
	ctx.wasmProgramUseOnce.Do(func() { ctx.wasmProgramUse = use })
	ctx.valueFormatPackages = make(map[*types.Package]bool)
	for _, rewrite := range plan.calls {
		ctx.valueFormatPackages[rewrite.owner] = true
	}
	if verbose {
		fmt.Fprintln(os.Stderr, "llgo: specialize default value formatting")
	}
}

// A modeled //export declaration is an extra RTA root. Other export
// directives and aliases in user code can enter Go outside the SSA call graph.
func formatExportRoots(prog *ssa.Program, roots []*ssa.Function, pkg *packages.Package, goroot string) ([]*ssa.Function, bool) {
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
