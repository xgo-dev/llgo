package build

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"os"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/types/typeutil"
)

type valueFormatPlan struct {
	callee *ssa.Function
	names  map[*types.Package]*typeutil.Map
	calls  []formatRewrite
}

type formatRewrite struct {
	call     *ssa.Call
	function *ssa.Function
	input    ssa.Value
	packed   []ssa.Instruction
	saved    ssa.CallCommon
}

func (ctx *context) valueFormatTarget() bool {
	return canSpecializeFormats(ctx) && ctx.buildConf.OptLevel == optlevel.Oz &&
		ctx.buildConf.BuildMode == BuildModeExe && ctx.crossCompile.WasmProfile == crosscompile.WasmProfileW32 &&
		ctx.crossCompile.WasmProvider == crosscompile.WasmProviderWASI
}

func (ctx *context) newValueFormatPlan() *valueFormatPlan {
	if ctx.frontend != nil || len(ctx.valueFormatFunctions) == 0 {
		return nil
	}
	pkg := ctx.progSSA.ImportedPackage("fmt")
	if pkg == nil || pkg.Func("Sprintf") == nil {
		return nil
	}
	return &valueFormatPlan{callee: pkg.Func("Sprintf"), names: ctx.valueFormatFunctions}
}

func (plan *valueFormatPlan) record(instruction ssa.Instruction, owner *ssa.Package) {
	call, ok := instruction.(*ssa.Call)
	if !ok || owner == nil || plan.names[owner.Pkg] == nil {
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
	values := formatValues(call, common.Args[1])
	if len(values) != 1 || values[0] == nil {
		return
	}
	box := values[0]
	if refs := box.Referrers(); refs == nil || len(*refs) != 1 {
		return
	}
	name, ok := plan.names[owner.Pkg].At(box.X.Type()).(string)
	if !ok {
		return
	}
	helper := owner.Func(name)
	if helper == nil {
		return
	}
	varargs := common.Args[1].(*ssa.Slice)
	array := varargs.X.(*ssa.Alloc)
	packed := []ssa.Instruction{box, array, varargs}
	for _, ref := range *array.Referrers() {
		if addr, ok := ref.(*ssa.IndexAddr); ok {
			packed = append(packed, addr, (*addr.Referrers())[0])
		}
	}
	plan.calls = append(plan.calls, formatRewrite{call: call, function: helper, input: box.X, packed: packed})
}

func formatDereference(t types.Type) bool {
	switch t.Underlying().(type) {
	case *types.Array, *types.Slice, *types.Struct, *types.Map:
		return true
	default:
		return false
	}
}

// defaultFormatType checks the actual paths visited by a default formatter.
// A named value can have methods that override formatting at any level.
func defaultFormatType(t types.Type, top bool, visiting map[types.Type]bool) bool {
	t = types.Unalias(t)
	if visiting[t] {
		return false
	}
	visiting[t] = true
	defer delete(visiting, t)
	methods := types.NewMethodSet(t)
	for i := 0; i < methods.Len(); i++ {
		switch methods.At(i).Obj().Name() {
		case "Format", "String", "Error":
			return false
		}
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		return u.Kind() >= types.Bool && u.Kind() <= types.UnsafePointer
	case *types.Array:
		return defaultFormatType(u.Elem(), false, visiting)
	case *types.Slice:
		return defaultFormatType(u.Elem(), false, visiting)
	case *types.Struct:
		for i := 0; i < u.NumFields(); i++ {
			f := u.Field(i)
			if !defaultFormatType(f.Type(), false, visiting) {
				return false
			}
		}
		return true
	case *types.Map:
		return defaultFormatType(u.Key(), false, visiting) && defaultFormatType(u.Elem(), false, visiting)
	case *types.Chan:
		return true
	case *types.Pointer:
		return !top || !formatDereference(u.Elem()) || defaultFormatType(u.Elem(), false, visiting)
	default:
		return false
	}
}

func (plan *valueFormatPlan) tentativeRewrite() (map[*ssa.BasicBlock][]ssa.Instruction, map[ssa.Instruction]bool) {
	remove := make(map[ssa.Instruction]bool)
	blocks := make(map[*ssa.BasicBlock][]ssa.Instruction)
	for i := range plan.calls {
		rewrite := &plan.calls[i]
		common := rewrite.call.Common()
		rewrite.saved = *common
		common.Value = rewrite.function
		common.Args = []ssa.Value{rewrite.input}
		for _, instruction := range rewrite.packed {
			remove[instruction] = true
			block := instruction.Block()
			blocks[block] = block.Instrs
		}
	}
	for block, original := range blocks {
		block.Instrs = slices.DeleteFunc(slices.Clone(original), func(instruction ssa.Instruction) bool {
			return remove[instruction]
		})
	}
	return blocks, remove
}

func (plan *valueFormatPlan) restore(blocks map[*ssa.BasicBlock][]ssa.Instruction) {
	for block, original := range blocks {
		block.Instrs = original
	}
	for _, rewrite := range plan.calls {
		*rewrite.call.Common() = rewrite.saved
	}
}

func removeFormatRef(value ssa.Value, instruction ssa.Instruction) {
	if refs := value.Referrers(); refs != nil {
		*refs = slices.DeleteFunc(*refs, func(ref ssa.Instruction) bool { return ref == instruction })
	}
}

func (plan *valueFormatPlan) commit(remove map[ssa.Instruction]bool) map[*types.Package]bool {
	owners := make(map[*types.Package]bool)
	for _, rewrite := range plan.calls {
		for _, arg := range rewrite.saved.Args {
			removeFormatRef(arg, rewrite.call)
		}
		if refs := rewrite.input.Referrers(); refs != nil {
			*refs = append(*refs, rewrite.call)
		}
		owners[rewrite.call.Parent().Pkg.Pkg] = true
	}
	for instruction := range remove {
		for _, operand := range instruction.Operands(nil) {
			if *operand != nil {
				removeFormatRef(*operand, instruction)
			}
		}
		if alloc, ok := instruction.(*ssa.Alloc); ok {
			fn := alloc.Parent()
			fn.Locals = slices.DeleteFunc(fn.Locals, func(local *ssa.Alloc) bool { return local == alloc })
		}
	}
	return owners
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
	blocks, remove := plan.tentativeRewrite()
	use := analyzeWasmProgramUse(ctx.progSSA, roots)
	for fn := range use.reachable {
		if fn.Pkg == plan.callee.Pkg && (fn.Name() == "printArg" || fn.Name() == "printValue") {
			plan.restore(blocks)
			if verbose {
				fmt.Fprintln(os.Stderr, "llgo: retain generic fmt: general formatting is reachable")
			}
			return
		}
	}
	ctx.wasmProgramUseOnce.Do(func() { ctx.wasmProgramUse = use })
	ctx.valueFormatPackages = plan.commit(remove)
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
