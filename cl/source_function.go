package cl

import (
	"go/ast"
	"go/types"

	"github.com/xgo-dev/llgo/internal/directive"
	"golang.org/x/tools/go/ssa"
)

// sourceFunction is a backend-local view of immutable declaration properties.
// LLVM functions continue to be created through the existing constructors;
// properties such as NoInline are applied after creation.
type functionProperties = directive.Function

type sourceFunction struct {
	*ssa.Function
	functionProperties
}

func (p *context) sourceFunction(fn *ssa.Function) *sourceFunction {
	if p.sourceFunctions == nil {
		p.sourceFunctions = make(map[*ssa.Function]*sourceFunction)
	}
	if f := p.sourceFunctions[fn]; f != nil {
		return f
	}
	f := &sourceFunction{Function: fn}
	if fn != nil {
		source := fn
		if origin := fn.Origin(); origin != nil {
			source = origin
		}
		obj, _ := source.Object().(*types.Func)
		var pkg *types.Package
		if source.Pkg != nil {
			pkg = source.Pkg.Pkg
		} else if obj != nil {
			pkg = obj.Pkg()
		}
		syntax, _ := source.Syntax().(*ast.FuncDecl)
		if syntax != nil || source.Synthetic == "" {
			var found bool
			f.functionProperties, found = p.prog.FunctionDirectives(pkg, obj, syntax)
			if !found && !p.options.PreloadedSyntax {
				f.functionProperties, _ = p.prog.Directives().LookupFunction(syntax)
			}
		}
	}
	p.sourceFunctions[fn] = f
	return f
}
