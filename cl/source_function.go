package cl

import (
	"go/ast"
	"go/types"

	"github.com/xgo-dev/llgo/internal/directive"
	llssa "github.com/xgo-dev/llgo/ssa"
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
	obj, _ := fn.Object().(*types.Func)
	if syntax == nil && fn.Synthetic != "" && (obj == nil || fn.Prog.FuncValue(obj) != fn) {
		return directive.Function{}
	}
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

func applyFunctionProperties(fn llssa.Function, properties directive.Function) {
	if properties.Cold {
		fn.SetCold()
	}
	if properties.NoReturn {
		fn.SetNoReturn()
	}
}

// A replaced body supplies the callable contract. Keep the original source
// properties separate: analyses of the original body still need its own flags.
func (p *context) applyFunctionAttributes(fn llssa.Function, source *ssa.Function) {
	properties := p.functionDirectives(source)
	origin := source
	if generic := source.Origin(); generic != nil {
		origin = generic
	}
	if obj, ok := origin.Object().(*types.Func); ok {
		if replacement, ok := p.patchedFunctionProperties(obj); ok {
			properties = replacement
		}
	}
	applyFunctionProperties(fn, properties)
	properties = properties.WithPositions(p.fset)
	if properties.ContractError != nil {
		panic(properties.ContractError)
	}
	fn.ApplyValueAttributes(source.Signature, properties.Values)
}

func (p *context) patchedFunctionProperties(obj *types.Func) (directive.Function, bool) {
	if obj.Pkg() != nil {
		if patch, ok := p.patches[llssa.PathOf(obj.Pkg())]; ok {
			if records := p.prog.PackageDirectives(patch.Types); records != nil {
				if r, ok := records.Objects[obj.Origin()].(*directive.FunctionDecl); ok {
					return r.Function, true
				}
				_, name := typesFuncName(llssa.PathOf(obj.Pkg()), obj)
				if r, ok := records.Names[name].(*directive.FunctionDecl); ok {
					return r.Function, true
				}
			}
		}
	}
	return directive.Function{}, false
}

// Backend-created entries consume prepared records without reopening sources.
func (p *context) initFunctionAttributes(fn llssa.Function, obj *types.Func, signature *types.Signature) {
	properties, ok := p.patchedFunctionProperties(obj)
	if !ok {
		properties, _ = p.prog.FunctionDirectives(obj.Pkg(), obj, nil)
	}
	applyFunctionProperties(fn, properties)
	properties = properties.WithPositions(p.fset)
	if properties.ContractError != nil {
		panic(properties.ContractError)
	}
	source := obj.Type().(*types.Signature)
	if source.Recv() != nil && signature.Recv() != nil && !types.Identical(source.Recv().Type(), signature.Recv().Type()) {
		// A pointer-receiver wrapper receives an address containing the source
		// value. Source receiver promises describe the loaded value, not its home.
		values := properties.Values[:0:0]
		for _, attr := range properties.Values {
			if attr.Target.Scope != directive.Receiver {
				values = append(values, attr)
			}
		}
		properties.Values = values
	}
	fn.ApplyValueAttributes(signature, properties.Values)
}
