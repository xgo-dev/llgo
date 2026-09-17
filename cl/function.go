package cl

import (
	"go/ast"
	"go/types"

	"github.com/xgo-dev/llgo/internal/exportdata"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// aFunction adds source properties to a Go SSA function. Backend functions are
// created separately and retain their own LLVM attributes.
type aFunction struct {
	*ssa.Function
	cold     bool
	noreturn bool
}

func (f *aFunction) setAttributes(attr exportdata.Function) {
	f.cold, f.noreturn = attr.Cold, attr.NoReturn
}

func (f *aFunction) applyAttributes(fn llssa.Function) {
	if f.cold {
		fn.SetCold()
	}
	if f.noreturn {
		fn.SetNoReturn()
	}
}

// function associates source properties when constructing the frontend wrapper.
// Further lowering passes the wrapper's values to the backend function.
func (p *context) function(fn *ssa.Function) aFunction {
	f := aFunction{Function: fn}
	origin := fn
	if generic := fn.Origin(); generic != nil {
		origin = generic
	}
	if decl, ok := origin.Syntax().(*ast.FuncDecl); ok {
		owner := p.goTyps
		if origin.Pkg != nil {
			owner = origin.Pkg.Pkg
		}
		_, name := astFuncName("", decl)
		f.setAttributes(p.exportedFunction(owner, name))
	} else if obj, ok := origin.Object().(*types.Func); ok && origin.Prog.FuncValue(obj) == origin {
		f.setAttributes(p.functionAttributes(obj))
	}
	return f
}

func (p *context) exportedFunction(pkg *types.Package, name string) exportdata.Function {
	if patch, ok := p.patches[llssa.PathOf(pkg)]; ok {
		pkg = patch.Types
	}
	return p.options.Exports.lookup(pkg, name)
}

func (p *context) functionAttributes(fn *types.Func) exportdata.Function {
	fn = fn.Origin()
	if fn.Pkg() == nil {
		return exportdata.Function{}
	}
	if recv := fn.Type().(*types.Signature).Recv(); recv != nil {
		if _, ok := recv.Type().Underlying().(*types.Interface); ok {
			return exportdata.Function{}
		}
	}
	_, name := typesFuncName("", fn)
	return p.exportedFunction(fn.Pkg(), name)
}

// Runtime and method entries constructed by the backend use the same package
// records even when their source Go SSA function is unavailable.
func (p *context) initFunctionAttributes(fn llssa.Function, source *types.Func) {
	var f aFunction
	f.setAttributes(p.functionAttributes(source))
	f.applyAttributes(fn)
}
