package cl

import (
	"go/ast"
	"go/types"

	"github.com/xgo-dev/llgo/internal/directive"
	"golang.org/x/tools/go/ssa"
)

// functionDirectives looks up prepared properties for a Go SSA function.
// Generic instances use their source declaration; synthetic wrappers without
// one do not inherit the wrapped function's directives.
func (p *context) functionDirectives(fn *ssa.Function) directive.Function {
	if fn == nil {
		return directive.Function{}
	}
	if origin := fn.Origin(); origin != nil {
		fn = origin
	}
	syntax, _ := fn.Syntax().(*ast.FuncDecl)
	if syntax == nil && fn.Synthetic != "" {
		return directive.Function{}
	}
	obj, _ := fn.Object().(*types.Func)
	var pkg *types.Package
	if fn.Pkg != nil {
		pkg = fn.Pkg.Pkg
	} else if obj != nil {
		pkg = obj.Pkg()
	}
	properties, found := p.prog.FunctionDirectives(pkg, obj, syntax)
	if !found && !p.options.PreloadedSyntax {
		properties, _ = p.prog.Directives().LookupFunction(syntax)
	}
	return properties
}
