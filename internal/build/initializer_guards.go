package build

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
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

// Select unused initializers at the final link. Each package archive defines
// the same guards for every executable in the selected size profile.
func configureOptionalInitializers(ctx *context, goroot string, inputs *linkPruningInputs) {
	conf := ctx.buildConf
	if !hostedValueFormatSupport(conf, ctx.crossCompile, ctx.frontendOptions) ||
		ctx.progSSA == nil || ctx.frontend != nil || len(conf.SourceOverlay) != 0 {
		return
	}
	browser := conf.Goarch == "wasm"
	candidates := make(map[*ssa.Global]*ssa.Store)
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
		case browser && pkg.PkgPath == "os":
			groups = optionalStandardFileInitializers(source.Func("init"))
		case !browser && hasStaticMapValues(pkg.Package):
			groups = cl.OptionalMapInitializers(source.Func("init"))
		}
		if len(groups) == 0 {
			continue
		}
		pkg.guardedInitializers = make(map[*ssa.BasicBlock][]cl.OptionalInitializer)
		for _, group := range groups {
			block := group.First.Block()
			pkg.guardedInitializers[block] = append(pkg.guardedInitializers[block], group)
			candidates[group.Global] = group.Store
		}
	}
	if len(candidates) == 0 {
		return
	}
	assembly := ctx.sizePruningRoots
	if browser {
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
	for fn := range rta.Analyze(roots, false).Reachable {
		for _, block := range fn.Blocks {
			for _, inst := range block.Instrs {
				if _, debug := inst.(*ssa.DebugRef); debug {
					continue
				}
				for _, operand := range inst.Operands(nil) {
					if global, ok := (*operand).(*ssa.Global); ok {
						if init, candidate := candidates[global]; candidate && init != inst {
							delete(candidates, global)
						}
					}
				}
			}
		}
	}
	for global := range candidates {
		ctx.initializerGuardsOff = append(ctx.initializerGuardsOff, cl.InitializerGuardName(global))
	}
	slices.Sort(ctx.initializerGuardsOff)
	if len(ctx.initializerGuardsOff) != 0 && (conf.Verbose || conf.PrintCommands) {
		kind := "map"
		if browser {
			kind = "standard file"
		}
		fmt.Fprintf(os.Stderr, "llgo: omit unused %s initializers (%d)\n", kind, len(ctx.initializerGuardsOff))
	}
}

// Map elements must come from static data. A pointer to an object with an
// observable lifetime, such as a registered finalizer, cannot be dropped.
func hasStaticMapValues(pkg *packages.Package) bool {
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
