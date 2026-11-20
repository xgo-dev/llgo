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

// Determine which initialized maps are visible to this executable. The
// package archive always defines the same guards for the native size profile.
func configureOptionalMapInitializers(ctx *context, goroot string) {
	conf := ctx.buildConf
	if conf.Goarch == "wasm" || !hostedValueFormatSupport(conf, ctx.crossCompile, ctx.frontendOptions) ||
		ctx.progSSA == nil || ctx.frontend != nil || len(conf.SourceOverlay) != 0 {
		return
	}
	candidates := make(map[*ssa.Global]*ssa.Store)
	for _, pkg := range ctx.pkgByID {
		if pkg.AltPkg != nil || !isStandardLibraryPackage(pkg.Package, goroot) || !hasStaticMapValues(pkg.Package) {
			continue
		}
		if source := ctx.progSSA.Package(pkg.Types); source != nil {
			for _, group := range cl.OptionalMapInitializers(source.Func("init")) {
				pkg.guardMapInitializers = true
				candidates[group.Global] = group.Store
			}
		}
	}
	if !ctx.nativeClosedProgram || len(candidates) == 0 {
		return
	}

	roots := wasmReflectRoots(ctx)
	blankImports := make(map[string]bool)
	for _, pkg := range ctx.pkgByID {
		roots = mapInitializerRoots(ctx, roots, pkg.Package, goroot, blankImports)
		if pkg.AltPkg != nil {
			roots = mapInitializerRoots(ctx, roots, pkg.AltPkg.Package, goroot, blankImports)
		}
	}
	roots = fmtspecialize.AppendSymbolRoots(ctx.progSSA, roots, ctx.sizePruningRoots)
	for global := range candidates {
		name := llssa.FullName(global.Pkg.Pkg, global.Name())
		if _, aliased := ctx.prog.Linkname(name); aliased || ctx.prog.HasLinknameTarget(name) ||
			blankImports[global.Pkg.Pkg.Path()] || slices.Contains(ctx.sizePruningRoots, name) {
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
		ctx.mapInitDrops = append(ctx.mapInitDrops, cl.MapInitializerGuardName(global))
	}
	slices.Sort(ctx.mapInitDrops)
	if len(ctx.mapInitDrops) != 0 && (conf.Verbose || conf.PrintCommands) {
		fmt.Fprintf(os.Stderr, "llgo: omit unused map initializers (%d)\n", len(ctx.mapInitDrops))
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

func mapInitializerRoots(ctx *context, roots []*ssa.Function, pkg *packages.Package, goroot string, blankImports map[string]bool) []*ssa.Function {
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
