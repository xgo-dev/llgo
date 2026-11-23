package typesynth

import (
	"go/ast"
	"go/types"

	"github.com/xgo-dev/llgo/internal/packages"
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

// PlanClosedCalls accepts a plan only when every use of an entry is a call
// handled by that plan. Keeping the generic implementation alongside a
// generated codec would retain most of the code the rewrite aims to remove.
func PlanClosedCalls[T any](pkg *packages.Package, provider *types.Package, entry func(string) bool, plan func(*ast.CallExpr, string) (T, bool)) ([]T, bool) {
	var planned []T
	seen := make(map[*ast.Ident]bool)
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || call.Ellipsis.IsValid() {
				return true
			}
			fn, id := Callee(pkg.TypesInfo, call.Fun)
			if fn != nil && fn.Pkg() == provider && entry(fn.Name()) {
				if item, ok := plan(call, fn.Name()); ok {
					seen[id] = true
					planned = append(planned, item)
				}
			}
			return true
		})
	}
	for id, obj := range pkg.TypesInfo.Uses {
		if fn, ok := obj.(*types.Func); ok && fn.Pkg() == provider && entry(fn.Name()) && !seen[id] {
			return nil, false
		}
	}
	return planned, true
}
