package build

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"os"
	"slices"
	"strconv"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/build/fmtspecialize"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
)

func wasiGuardedStaticData(ctx *context) bool {
	conf := ctx.buildConf
	return !conf.disableSizeSpecialization && sizeCodecBuild(conf, ctx.frontendOptions) &&
		wasiSizePruningConfig(conf, ctx.crossCompile, ctx.frontendOptions)
}

// Select unused initializers at the final link. Each package archive defines
// the same guards for every executable in the selected size profile.
func configureOptionalInitializers(ctx *context, goroot string, inputs *linkPruningInputs) {
	conf := ctx.buildConf
	hosted := hostedValueFormatSupport(conf, ctx.crossCompile, ctx.frontendOptions)
	if (!hosted && !wasiGuardedStaticData(ctx)) ||
		ctx.progSSA == nil || ctx.frontend != nil || len(conf.SourceOverlay) != 0 {
		return
	}
	wasm := conf.Goarch == "wasm"
	var errorConstructor *ssa.Function
	if wasm {
		errorConstructor = literalErrorConstructor(ctx, goroot)
	}
	candidates := make(map[*ssa.Global]cl.OptionalInitializer)
	sliceOwners := make(map[*ssa.Package]bool)
	for _, pkg := range ctx.pkgByID {
		if pkg.AltPkg != nil || !isStandardLibraryPackage(pkg.Package, goroot) {
			continue
		}
		source := ctx.progSSA.Package(pkg.Types)
		if source == nil {
			continue
		}
		var groups []cl.OptionalInitializer
		switch {
		case wasm && hosted && pkg.PkgPath == "os":
			groups = optionalStandardFileInitializers(source.Func("init"))
		case wasm && hasStaticDataValues(pkg.Package):
			groups = cl.OptionalSliceInitializers(source.Func("init"))
			if len(groups) != 0 {
				sliceOwners[source] = true
			}
		case !wasm && hasStaticDataValues(pkg.Package):
			groups = cl.OptionalMapInitializers(source.Func("init"))
		case wasm:
			groups = optionalLiteralInitializers(source.Func("init"), errorConstructor)
		}
		if len(groups) == 0 {
			continue
		}
		pkg.guardedInitializers = make(map[*ssa.BasicBlock][]cl.OptionalInitializer)
		for _, group := range groups {
			block := group.First.Block()
			pkg.guardedInitializers[block] = append(pkg.guardedInitializers[block], group)
			candidates[group.Global] = group
		}
	}
	if len(candidates) == 0 {
		return
	}
	assembly := ctx.sizePruningRoots
	if wasm {
		if !ctx.wasmClosedProgram || inputs == nil || inputs.opaqueAsmPkg != "" ||
			len(ctx.initial) != 1 || hasOpaqueFormatExports(ctx, goroot) {
			return
		}
		assembly = inputs.roots
	} else if !ctx.nativeClosedProgram {
		return
	}

	roots := wasmReflectRoots(ctx)
	blankImports := make(map[string]bool)
	for _, pkg := range ctx.pkgByID {
		roots = guardedInitializerRoots(ctx, roots, pkg.Package, goroot, blankImports)
		if pkg.AltPkg != nil {
			roots = guardedInitializerRoots(ctx, roots, pkg.AltPkg.Package, goroot, blankImports)
		}
	}
	roots = fmtspecialize.AppendSymbolRoots(ctx.progSSA, roots, assembly)
	var keepErrorConstructors bool
	if errorConstructor != nil {
		const symbol = "errors.New"
		_, aliased := ctx.prog.Linkname(symbol)
		keepErrorConstructors = aliased || ctx.prog.HasLinknameTarget(symbol) || slices.Contains(assembly, symbol)
	}
	// A literal slice can keep a shared object alive even when nobody reads
	// the slice. Allow omission only when the owner's data is private to its
	// variable initializers, including reads through another package or alias.
	observedData := make(map[*ssa.Package]bool)
	for owner := range sliceOwners {
		for _, member := range owner.Members {
			if global, ok := member.(*ssa.Global); ok {
				name := llssa.FullName(owner.Pkg, global.Name())
				if _, aliased := ctx.prog.Linkname(name); aliased || ctx.prog.HasLinknameTarget(name) || slices.Contains(assembly, name) {
					observedData[owner] = true
				}
			}
		}
	}
	for global := range candidates {
		name := llssa.FullName(global.Pkg.Pkg, global.Name())
		if _, aliased := ctx.prog.Linkname(name); aliased || ctx.prog.HasLinknameTarget(name) ||
			blankImports[global.Pkg.Pkg.Path()] || slices.Contains(assembly, name) {
			delete(candidates, global)
		}
	}
	if len(roots) == 0 || len(candidates) == 0 {
		return
	}
	unused := unobservedInitializers(roots, candidates, sliceOwners, observedData, nil)
	var callbackInitializers []cl.OptionalInitializer
	for _, group := range unused {
		if _, ok := group.Store.Val.(*ssa.UnOp); ok {
			callbackInitializers = append(callbackInitializers, group)
		}
	}
	// Address-taken callbacks can keep unrelated package data reachable through
	// indirect calls. Recheck after removing the already proven-unused edges.
	if len(callbackInitializers) != 0 {
		maps.Copy(unused, unobservedInitializers(roots, candidates, sliceOwners, observedData, callbackInitializers))
	}
	var dataCount, errorCount, interfaceCount, callbackCount, otherCount int
	for global, group := range unused {
		store := group.Store
		call, _ := store.Val.(*ssa.Call)
		errorValue := call != nil && errorConstructor != nil && call.Common().StaticCallee() == errorConstructor
		_, interfaceValue := store.Val.(*ssa.MakeInterface)
		_, callbackValue := store.Val.(*ssa.UnOp)
		if errorValue && keepErrorConstructors {
			continue
		}
		ctx.initializerGuardsOff = append(ctx.initializerGuardsOff, cl.InitializerGuardName(global))
		switch {
		case sliceOwners[global.Pkg]:
			dataCount++
		case errorValue:
			errorCount++
		case interfaceValue:
			interfaceCount++
		case callbackValue:
			callbackCount++
		default:
			otherCount++
		}
	}
	slices.Sort(ctx.initializerGuardsOff)
	if conf.Verbose || conf.PrintCommands {
		if dataCount != 0 {
			fmt.Fprintf(os.Stderr, "llgo: omit unused static slice initializers (%d)\n", dataCount)
		}
		if errorCount != 0 {
			fmt.Fprintf(os.Stderr, "llgo: omit unused literal error initializers (%d)\n", errorCount)
		}
		if interfaceCount != 0 {
			fmt.Fprintf(os.Stderr, "llgo: omit unused constant interface initializers (%d)\n", interfaceCount)
		}
		if callbackCount != 0 {
			fmt.Fprintf(os.Stderr, "llgo: omit unused static callback initializers (%d)\n", callbackCount)
		}
		if otherCount != 0 {
			kind := "map"
			if wasm {
				kind = "standard file"
			}
			fmt.Fprintf(os.Stderr, "llgo: omit unused %s initializers (%d)\n", kind, otherCount)
		}
	}
}

func unobservedInitializers(roots []*ssa.Function, candidates map[*ssa.Global]cl.OptionalInitializer,
	sliceOwners, knownData map[*ssa.Package]bool, omitted []cl.OptionalInitializer) map[*ssa.Global]cl.OptionalInitializer {
	if len(omitted) != 0 {
		restore := omitInitializerEdges(omitted)
		defer restore()
	}
	unused := maps.Clone(candidates)
	observedData := maps.Clone(knownData)
	for fn := range rta.Analyze(roots, false).Reachable {
		for _, block := range fn.Blocks {
			for _, inst := range block.Instrs {
				if _, debug := inst.(*ssa.DebugRef); debug {
					continue
				}
				for _, operand := range inst.Operands(nil) {
					if global, ok := (*operand).(*ssa.Global); ok {
						if sliceOwners[global.Pkg] && fn != global.Pkg.Func("init") {
							observedData[global.Pkg] = true
						}
						if init, candidate := unused[global]; candidate && init.Store != inst {
							delete(unused, global)
						}
					}
				}
			}
		}
	}
	maps.DeleteFunc(unused, func(global *ssa.Global, _ cl.OptionalInitializer) bool { return observedData[global.Pkg] })
	return unused
}

// Hide initializer bodies from reachability only. The coordinator restores the
// original SSA before package compilation, keeping archive contents reusable.
func omitInitializerEdges(omitted []cl.OptionalInitializer) func() {
	ends := make(map[*ssa.BasicBlock]map[ssa.Instruction]ssa.Instruction)
	for _, group := range omitted {
		block := group.First.Block()
		if ends[block] == nil {
			ends[block] = make(map[ssa.Instruction]ssa.Instruction)
		}
		ends[block][group.First] = group.Store
	}
	original := make(map[*ssa.BasicBlock][]ssa.Instruction, len(ends))
	for block, groups := range ends {
		original[block] = block.Instrs
		kept := make([]ssa.Instruction, 0, len(block.Instrs))
		var end ssa.Instruction
		for _, inst := range block.Instrs {
			if last := groups[inst]; last != nil {
				end = last
			}
			if end == nil {
				kept = append(kept, inst)
			} else if end == inst {
				end = nil
			}
		}
		block.Instrs = kept
	}
	return func() {
		for block, instructions := range original {
			block.Instrs = instructions
		}
	}
}

// Container elements must come from static data. A constructor could return
// an object whose lifetime or initialization effects are observable.
func hasStaticDataValues(pkg *packages.Package) bool {
	if pkg.Types == nil || pkg.TypesInfo == nil || len(pkg.Types.Imports()) != 0 ||
		len(pkg.OtherFiles) != 0 || len(pkg.EmbedFiles) != 0 || len(pkg.TypesInfo.InitOrder) == 0 {
		return false
	}
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "init" {
				return false
			}
		}
	}
	for _, init := range pkg.TypesInfo.InitOrder {
		if !staticDataExpr(pkg, init.Rhs) {
			return false
		}
	}
	return true
}

func staticDataExpr(pkg *packages.Package, expr ast.Expr) bool {
	if pkg.TypesInfo.Types[expr].Value != nil {
		return true
	}
	switch expr := expr.(type) {
	case *ast.Ident:
		obj := pkg.TypesInfo.Uses[expr]
		_, variable := obj.(*types.Var)
		return obj == types.Universe.Lookup("nil") || variable && obj.Pkg() == pkg.Types
	case *ast.ParenExpr:
		return staticDataExpr(pkg, expr.X)
	case *ast.UnaryExpr:
		literal, ok := expr.X.(*ast.CompositeLit)
		return expr.Op == token.AND && ok && staticDataExpr(pkg, literal)
	case *ast.CompositeLit:
		for _, element := range expr.Elts {
			if !staticDataExpr(pkg, element) {
				return false
			}
		}
		return true
	case *ast.KeyValueExpr:
		return staticDataExpr(pkg, expr.Key) && staticDataExpr(pkg, expr.Value)
	}
	return false
}

func guardedInitializerRoots(ctx *context, roots []*ssa.Function, pkg *packages.Package, goroot string, blankImports map[string]bool) []*ssa.Function {
	if pkg == nil || pkg.Types == nil {
		return roots
	}
	if p := ctx.progSSA.Package(pkg.Types); p != nil {
		if init := p.Func("init"); init != nil {
			roots = append(roots, init)
		}
		for _, member := range p.Members {
			if fn, ok := member.(*ssa.Function); ok && len(fn.Blocks) != 0 {
				name := llssa.FullName(pkg.Types, fn.Name())
				if _, aliased := ctx.prog.Linkname(name); aliased || ctx.prog.HasLinknameTarget(name) {
					roots = append(roots, fn)
				}
			}
		}
	}
	for _, file := range pkg.Syntax {
		for _, imp := range file.Imports {
			if imp.Name != nil && imp.Name.Name == "_" {
				path, _ := strconv.Unquote(imp.Path.Value)
				blankImports[path] = true
			}
		}
	}
	roots, _ = formatExportRoots(ctx.progSSA, roots, pkg, goroot)
	return roots
}
