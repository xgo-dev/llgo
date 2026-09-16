package cl

import (
	"go/ast"
	"go/types"

	"github.com/xgo-dev/llgo/internal/directive"
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

func (f *aFunction) readAttributes(doc *ast.CommentGroup) {
	for _, attr := range directive.ParseGroup(doc) {
		switch attr.Name {
		case "llgo:cold":
			f.cold = true
		case "llgo:noreturn":
			f.noreturn = true
		}
	}
}

func (f *aFunction) applyAttributes(fn llssa.Function) {
	if f.cold {
		fn.SetCold()
	}
	if f.noreturn {
		fn.SetNoReturn()
	}
}

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
		name, _ := astFuncName(llssa.PathOf(owner), decl)
		doc := decl.Doc
		if patched, ok := p.patchFunctionAttributeSource(owner, name); ok {
			doc = patched
		}
		f.readAttributes(doc)
	} else if obj, ok := origin.Object().(*types.Func); ok && origin.Prog.FuncValue(obj) == origin {
		// Export-only functions have no syntax and are marked synthetic by
		// x/tools. Match the declaration itself, excluding method wrappers.
		f.readAttributes(p.functionAttributeSource(obj))
	}
	return f
}

func (p *context) patchFunctionAttributeSource(pkg *types.Package, name string) (*ast.CommentGroup, bool) {
	if patch, ok := p.patches[llssa.PathOf(pkg)]; ok {
		return p.options.FunctionAttributes.lookup(patch.Types, name)
	}
	return nil, false
}

func (p *context) functionAttributeSource(fn *types.Func) *ast.CommentGroup {
	fn = fn.Origin()
	if fn.Pkg() == nil {
		return nil
	}
	if recv := fn.Type().(*types.Signature).Recv(); recv != nil {
		// Promoted interface methods have no function declaration. In
		// particular, an interface alias can have an unnamed receiver type.
		if _, ok := recv.Type().Underlying().(*types.Interface); ok {
			return nil
		}
	}
	name, _ := typesFuncName(llssa.PathOf(fn.Pkg()), fn)
	if doc, ok := p.patchFunctionAttributeSource(fn.Pkg(), name); ok {
		return doc
	}
	doc, _ := p.options.FunctionAttributes.lookup(fn.Pkg(), name)
	return doc
}

// Backend-created runtime and method entries can precede source body lowering.
// They use the same source lookup without making ssa depend on the frontend.
func (p *context) initFunctionAttributes(fn llssa.Function, source *types.Func) {
	var f aFunction
	f.readAttributes(p.functionAttributeSource(source))
	f.applyAttributes(fn)
}
