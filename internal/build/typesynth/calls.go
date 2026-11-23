package typesynth

import (
	"go/ast"
	"go/types"
)

// Callee identifies a direct call to a package-level function.
func Callee(info *types.Info, expr ast.Expr) (*types.Func, *ast.Ident) {
	var id *ast.Ident
	switch expr := ast.Unparen(expr).(type) {
	case *ast.Ident:
		id = expr
	case *ast.SelectorExpr:
		if info.Selections[expr] == nil {
			id = expr.Sel
		}
	}
	fn, _ := info.Uses[id].(*types.Func)
	return fn, id
}
