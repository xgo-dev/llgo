package cl

import (
	"go/ast"
	"go/token"
	"go/types"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// aFunction carries source information alongside a frontend function. Two
// source functions can refer to the same backend symbol without sharing their
// declaration metadata. The backend entry is created separately, on demand.
type aFunction struct {
	*ssa.Function
	declaration *llssa.FunctionDeclaration
	impl        llssa.Function
}

func (p *context) function(fn *ssa.Function) *aFunction {
	if p.funcs == nil {
		p.funcs = make(map[*ssa.Function]*aFunction)
	}
	ret := p.funcs[fn]
	if ret == nil {
		ret = &aFunction{Function: fn}
		p.funcs[fn] = ret
	}
	if ret.declaration == nil {
		ret.declaration = p.functionDeclaration(fn)
	}
	return ret
}

func (p *context) functionDeclaration(fn *ssa.Function) *llssa.FunctionDeclaration {
	if origin := fn.Origin(); origin != nil {
		fn = origin
	}
	if decl, ok := fn.Syntax().(*ast.FuncDecl); ok {
		owner := p.goTyps
		if fn.Pkg != nil {
			owner = fn.Pkg.Pkg
		}
		name, _ := astFuncName(llssa.PathOf(owner), decl)
		fset := p.fset
		if fn.Prog != nil {
			fset = fn.Prog.Fset
		}
		return p.prog.SourceFunctionDeclaration(owner, fset, name, decl.Pos()).Effective()
	}
	// Synthetic wrappers have their own calling convention and declaration
	// identity; an underlying method's source properties do not describe them.
	if fn.Synthetic == "" {
		if obj, ok := fn.Object().(*types.Func); ok {
			return p.prog.FunctionDeclarationOf(obj)
		}
	}
	return nil
}

// BindPackageFunctionDeclarations selects the source functions supplied by an
// enabled package patch. This runs after syntax preparation and before any
// concurrent backend consumes the declaration objects.
func BindPackageFunctionDeclarations(prog llssa.Program, original, patched *types.Package, fset *token.FileSet, files []*ast.File) {
	for _, file := range files {
		for _, node := range file.Decls {
			if decl, ok := node.(*ast.FuncDecl); ok {
				if decl.Recv == nil && decl.Name.Name == "init" {
					continue
				}
				name, _ := astFuncName(llssa.PathOf(patched), decl)
				replacement := prog.SourceFunctionDeclaration(patched, fset, name, decl.Pos())
				if replacement == nil {
					continue
				}
				prog.ReplaceFunctionDeclarations(patched, replacement)
				if original != nil {
					prog.ReplaceFunctionDeclarations(original, replacement)
				}
			}
		}
	}
}
