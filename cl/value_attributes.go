package cl

import (
	"go/ast"
	"go/types"
)

// The syntax preload can use an export-data-only package, where private
// functions are absent. Their contracts are still parsed and are type checked
// when a definition or caller-side declaration is lowered.
func sourceAttributeFunction(pkg *types.Package, decl *ast.FuncDecl) *types.Func {
	if decl.Recv == nil {
		fn, _ := pkg.Scope().Lookup(decl.Name.Name).(*types.Func)
		return fn
	}
	if len(decl.Recv.List) != 1 {
		return nil
	}
	recv := decl.Recv.List[0].Type
	if p, ok := recv.(*ast.StarExpr); ok {
		recv = p.X
	}
	obj, ok := pkg.Scope().Lookup(recvTypeName(recv)).(*types.TypeName)
	if !ok {
		return nil
	}
	named, ok := types.Unalias(obj.Type()).(*types.Named)
	if !ok {
		return nil
	}
	for i := 0; i < named.NumMethods(); i++ {
		if method := named.Method(i); method.Name() == decl.Name.Name {
			return method
		}
	}
	return nil
}
