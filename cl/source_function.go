package cl

import (
	"go/ast"
	"go/types"

	"github.com/xgo-dev/llgo/internal/directive"
	llssa "github.com/xgo-dev/llgo/ssa"
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
		if syntax != nil || source.Synthetic == "" || obj != nil && source.Prog.FuncValue(obj) == source {
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

func applyFunctionProperties(fn llssa.Function, properties functionProperties) {
	if properties.Cold {
		fn.SetCold()
	}
	if properties.NoReturn {
		fn.SetNoReturn()
	}
}

// A replaced body supplies the callable contract. Keep sourceFunction's source
// properties separate: analyses of the original body still need its own flags.
func (p *context) applyFunctionAttributes(fn llssa.Function, source *ssa.Function) {
	properties := p.sourceFunction(source).functionProperties
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

func (p *context) patchedFunctionProperties(obj *types.Func) (functionProperties, bool) {
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
	return functionProperties{}, false
}

// Backend-created entries consume prepared records without reopening sources.
func (p *context) initFunctionAttributes(fn llssa.Function, obj *types.Func) {
	properties, ok := p.patchedFunctionProperties(obj)
	if !ok {
		properties, _ = p.prog.FunctionDirectives(obj.Pkg(), obj, nil)
	}
	applyFunctionProperties(fn, properties)
	properties = properties.WithPositions(p.fset)
	if properties.ContractError != nil {
		panic(properties.ContractError)
	}
	signature := fn.Type.RawType().(*types.Signature)
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
