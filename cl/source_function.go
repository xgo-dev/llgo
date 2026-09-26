package cl

import (
	"go/ast"
	"go/types"

	"github.com/xgo-dev/llgo/internal/directive"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// sourceFunction keeps the concrete SSA body/signature with its source record.
// Generic instances share their origin's declaration. Patch selection belongs
// to call target resolution and never changes this association.
type sourceFunction struct {
	SSA  *ssa.Function
	Decl *directive.FunctionDecl
}

func (p *context) sourceFunction(fn *ssa.Function) sourceFunction {
	if source, ok := p.sourceFunctions[fn]; ok {
		return source
	}
	source := sourceFunction{SSA: fn}
	if fn != nil {
		if origin := fn.Origin(); origin != nil {
			fn = origin
		}
		syntax, _ := fn.Syntax().(*ast.FuncDecl)
		if syntax != nil || fn.Synthetic == "" {
			obj, _ := fn.Object().(*types.Func)
			var pkg *types.Package
			if fn.Pkg != nil {
				pkg = fn.Pkg.Pkg
			} else if obj != nil {
				pkg = obj.Pkg()
			}
			source.Decl = p.prog.FunctionDeclaration(pkg, obj, syntax)
			if source.Decl == nil && !p.options.PreloadedSyntax {
				source.Decl = p.prog.Directives().FunctionDeclaration(syntax)
			}
		}
	}
	if p.sourceFunctions == nil {
		p.sourceFunctions = make(map[*ssa.Function]sourceFunction)
	}
	p.sourceFunctions[source.SSA] = source
	return source
}

// callableDeclaration selects a patch record only for packages with patches.
// Backend-created entries also use this selection without constructing Go SSA.
func (p *context) callableDeclaration(pkg *types.Package, obj *types.Func, name string, source *directive.FunctionDecl) *directive.FunctionDecl {
	if patch, ok := p.patches[llssa.PathOf(pkg)]; ok {
		if records := p.prog.PackageDirectives(patch.Types); records != nil {
			if obj != nil {
				if decl, ok := records.Objects[obj.Origin()].(*directive.FunctionDecl); ok {
					return decl
				}
			}
			if decl, ok := records.Names[name].(*directive.FunctionDecl); ok {
				return decl
			}
		}
	}
	return source
}
