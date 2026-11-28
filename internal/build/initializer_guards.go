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
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
)

func wasiGuardedStaticData(conf *Config, target crosscompile.Export, frontend cl.Options) bool {
	return sizeCodecBuild(conf, frontend) && wasiSizePruningConfig(conf, target, frontend)
}

// Select unused initializers at the final link. Each package archive defines
// the same guards for every executable in the selected size profile.
func configureOptionalInitializers(ctx *context, goroot string, inputs *linkPruningInputs) {
	conf := ctx.buildConf
	hosted := hostedValueFormatSupport(conf, ctx.crossCompile, ctx.frontendOptions)
	embedded := embeddedSizePruningConfig(conf, ctx.crossCompile, ctx.frontendOptions)
	wasi := wasiGuardedStaticData(conf, ctx.crossCompile, ctx.frontendOptions)
	if (!hosted && !embedded && !wasi) ||
		ctx.progSSA == nil || ctx.frontend != nil || len(conf.SourceOverlay) != 0 {
		return
	}
	wasm := conf.Goarch == "wasm"
	native := hosted && !wasm
	constructors := findLiteralConstructors(ctx, goroot)
	candidates := make(map[*ssa.Global]cl.OptionalInitializer)
	sliceOwners := make(map[*ssa.Package]bool)
	var standardFiles []cl.OptionalInitializer
	var darwinOS *ssa.Package
	for _, pkg := range ctx.pkgByID {
		if pkg.AltPkg != nil || !isStandardLibraryPackage(pkg.Package, goroot) {
			continue
		}
		source := ctx.progSSA.Package(pkg.Types)
		if source == nil {
			continue
		}
		if ctx.nativeClosedProgram && conf.Goos == "darwin" && pkg.PkgPath == "os" &&
			source.Func(darwinExecutableCwdInitializer) != nil {
			darwinOS = source
		}
		init := source.Func("init")
		var groups []cl.OptionalInitializer
		switch {
		case (hosted || wasi || embedded) && pkg.PkgPath == "os":
			groups = optionalStandardFileInitializers(init)
			if wasi || embedded || native {
				if len(groups) != 3 {
					groups = nil
				}
				standardFiles = groups
			}
			if embedded || native {
				groups = orderedOptionalInitializers(init, groups, optionalLiteralInitializers(init, constructors))
			}
		case hasStaticDataValues(pkg.Package):
			groups = cl.OptionalSliceInitializers(init)
		default:
			groups = optionalLiteralInitializers(init, constructors)
		}
		// Literal maps can be guarded independently of the package's other
		// initialization, including explicit init functions.
		groups = orderedOptionalInitializers(init, cl.OptionalMapInitializers(init), groups)
		if len(groups) == 0 {
			continue
		}
		pkg.guardedInitializers = make(map[*ssa.BasicBlock][]cl.OptionalInitializer)
		for _, group := range groups {
			block := group.First.Block()
			pkg.guardedInitializers[block] = append(pkg.guardedInitializers[block], group)
			candidates[group.Global] = group
			if _, slice := group.Store.Val.(*ssa.Slice); slice {
				sliceOwners[source] = true
			}
		}
	}
	if len(candidates) == 0 && darwinOS == nil {
		return
	}
	assembly := ctx.sizePruningRoots
	if wasm || embedded {
		closed := wasm && ctx.wasmClosedProgram || embedded && ctx.embeddedClosedProgram
		if !closed || inputs == nil || inputs.opaqueAsmPkg != "" ||
			len(ctx.initial) != 1 || hasOpaqueFormatExports(ctx, goroot) {
			return
		}
		assembly = inputs.roots
	} else if !ctx.nativeClosedProgram {
		return
	}

	symbolRoots := make(map[string]bool)
	for name, target := range ctx.prog.Linknames() {
		symbolRoots[name], symbolRoots[target] = true, true
	}
	for _, name := range assembly {
		symbolRoots[name] = true
	}
	isSymbolRoot := func(name string) bool { return symbolRoots[name] }
	roots := wasmReflectRoots(ctx)
	blankImports := make(map[string]bool)
	for _, pkg := range ctx.pkgByID {
		roots = guardedInitializerRoots(ctx, roots, pkg.Package, goroot, blankImports, isSymbolRoot)
		if pkg.AltPkg != nil {
			roots = guardedInitializerRoots(ctx, roots, pkg.AltPkg.Package, goroot, blankImports, isSymbolRoot)
		}
	}
	roots = appendSymbolRoots(ctx.progSSA, roots, assembly)
	// A literal slice can keep a shared object alive even when nobody reads
	// the slice. Allow omission only when the owner's data is private to its
	// variable initializers, including reads through another package or alias.
	observedData := make(map[*ssa.Package]bool)
	for owner := range sliceOwners {
		for _, member := range owner.Members {
			if global, ok := member.(*ssa.Global); ok {
				name := llssa.FullName(owner.Pkg, global.Name())
				if isSymbolRoot(name) {
					observedData[owner] = true
				}
			}
		}
	}
	for global, group := range candidates {
		name := llssa.FullName(global.Pkg.Pkg, global.Name())
		if isSymbolRoot(name) || blankImports[global.Pkg.Pkg.Path()] {
			delete(candidates, global)
			continue
		}
		if call, ok := group.Store.Val.(*ssa.Call); ok {
			// A symbol reference can reach or replace the selected constructor.
			callee := call.Common().StaticCallee()
			origin := initializerFunctionOrigin(callee)
			if isSymbolRoot(llssa.FullName(origin.Pkg.Pkg, origin.Name())) ||
				origin != callee && isSymbolRoot(llssa.FullName(origin.Pkg.Pkg, callee.Name())) {
				delete(candidates, global)
			}
		}
	}
	if len(roots) == 0 {
		return
	}
	unused, _ := unobservedInitializers(roots, candidates, sliceOwners, observedData, nil)

	var callbackInitializers []cl.OptionalInitializer
	for _, group := range unused {
		if initializerStoresCallback(group.Store.Val) {
			callbackInitializers = append(callbackInitializers, group)
		}
	}
	// Address-taken callbacks can keep unrelated package data reachable through
	// indirect calls. Recheck after removing the already proven-unused edges.
	if darwinOS != nil && !blankImports["os"] {
		init := darwinOS.Func(darwinExecutableCwdInitializer)
		blocks := init.Blocks
		init.Blocks = nil
		withoutStartup, reachable := unobservedInitializers(roots, candidates, sliceOwners, observedData, callbackInitializers)
		init.Blocks = blocks
		// The additional drops are sound only when the omitted cwd edge
		// really disappears in the linked executable.
		if configureDarwinExecutableCwd(ctx, darwinOS, reachable, isSymbolRoot) {
			maps.Copy(unused, withoutStartup)
		}
	} else if len(callbackInitializers) != 0 {
		withoutCallbacks, _ := unobservedInitializers(roots, candidates, sliceOwners, observedData, callbackInitializers)
		maps.Copy(unused, withoutCallbacks)
	}
	// Drop the standard file startup group together. Keeping a single standard file
	// can retain the shared descriptor construction path.
	if slices.ContainsFunc(standardFiles, func(group cl.OptionalInitializer) bool {
		_, ok := unused[group.Global]
		return !ok
	}) {
		for _, group := range standardFiles {
			delete(unused, group.Global)
		}
	}
	var dataCount, errorCount, interfaceCount, callbackCount, mapCount, filesCount int
	for global, group := range unused {
		store := group.Store
		call, _ := store.Val.(*ssa.Call)
		errorValue := call != nil && constructors.newError != nil && call.Common().StaticCallee() == constructors.newError
		_, interfaceValue := store.Val.(*ssa.MakeInterface)
		_, sliceValue := store.Val.(*ssa.Slice)
		_, mapValue := store.Val.(*ssa.MakeMap)
		ctx.initializerGuardsOff = append(ctx.initializerGuardsOff, cl.InitializerGuardName(global))
		switch {
		case sliceValue:
			dataCount++
		case errorValue:
			errorCount++
		case interfaceValue:
			interfaceCount++
		case initializerStoresCallback(store.Val):
			callbackCount++
		case mapValue:
			mapCount++
		default:
			filesCount++
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
		if mapCount != 0 {
			fmt.Fprintf(os.Stderr, "llgo: omit unused map initializers (%d)\n", mapCount)
		}
		if filesCount != 0 {
			fmt.Fprintf(os.Stderr, "llgo: omit unused standard file initializers (%d)\n", filesCount)
		}
	}
}

func initializerStoresCallback(value ssa.Value) bool {
	switch value := value.(type) {
	case *ssa.UnOp:
		return true
	case *ssa.Call:
		_, ok := value.Type().Underlying().(*types.Signature)
		return ok
	}
	return false
}

func unobservedInitializers(roots []*ssa.Function, candidates map[*ssa.Global]cl.OptionalInitializer,
	sliceOwners, knownData map[*ssa.Package]bool, omitted []cl.OptionalInitializer) (map[*ssa.Global]cl.OptionalInitializer, *rta.Result) {
	if len(omitted) != 0 {
		restore := omitInitializerEdges(omitted)
		defer restore()
	}
	unused := maps.Clone(candidates)
	observedData := maps.Clone(knownData)
	reachable := rta.Analyze(roots, false)
	var operands [8]*ssa.Value
	for fn := range reachable.Reachable {
		for _, block := range fn.Blocks {
			for _, inst := range block.Instrs {
				if _, debug := inst.(*ssa.DebugRef); debug {
					continue
				}
				for _, operand := range inst.Operands(operands[:0]) {
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
	maps.DeleteFunc(unused, func(global *ssa.Global, group cl.OptionalInitializer) bool {
		_, slice := group.Store.Val.(*ssa.Slice)
		return slice && observedData[global.Pkg]
	})
	return unused, reachable
}

// The recognizers return disjoint ranges. Restore instruction order so
// lowering can emit guards from both groups in a shared block.
func orderedOptionalInitializers(init *ssa.Function, first, second []cl.OptionalInitializer) []cl.OptionalInitializer {
	if len(first) == 0 || len(second) == 0 {
		return append(first, second...)
	}
	starts := make(map[ssa.Instruction]cl.OptionalInitializer, len(first)+len(second))
	for _, group := range append(first, second...) {
		starts[group.First] = group
	}
	groups := make([]cl.OptionalInitializer, 0, len(starts))
	for _, block := range init.Blocks {
		for _, instr := range block.Instrs {
			if group, ok := starts[instr]; ok {
				groups = append(groups, group)
			}
		}
	}
	return groups
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

func guardedInitializerRoots(ctx *context, roots []*ssa.Function, pkg *packages.Package, goroot string, blankImports map[string]bool, isSymbolRoot func(string) bool) []*ssa.Function {
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
				if isSymbolRoot(name) {
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
