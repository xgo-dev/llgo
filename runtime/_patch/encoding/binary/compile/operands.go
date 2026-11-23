package compile

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/xgo-dev/llgo/internal/build/typesynth"
)

// A slice of an array has fixed bounds when both indices are constant. The
// argument is still evaluated, including any nil-pointer and bounds checks.
func arraySliceLength(info *types.Info, expr ast.Expr) int64 {
	slice, ok := ast.Unparen(expr).(*ast.SliceExpr)
	if !ok {
		return -1
	}
	typ := info.TypeOf(slice.X)
	if typ == nil {
		return -1
	}
	base := typ.Underlying()
	if ptr, ok := base.(*types.Pointer); ok {
		base = ptr.Elem().Underlying()
	}
	array, ok := base.(*types.Array)
	if !ok {
		return -1
	}
	lo, hi := int64(0), array.Len()
	for i, index := range []ast.Expr{slice.Low, slice.High} {
		if index == nil {
			continue
		}
		value := info.Types[index].Value
		if value == nil || value.Kind() != constant.Int {
			return -1
		}
		n, ok := constant.Int64Val(value)
		if !ok {
			return -1
		}
		if i == 0 {
			lo = n
		} else {
			hi = n
		}
	}
	if lo < 0 || hi < lo || hi > array.Len() {
		return -1
	}
	return hi - lo
}

func nonNilPointer(info *types.Info, expr ast.Expr) bool {
	switch expr := ast.Unparen(expr).(type) {
	case *ast.UnaryExpr:
		_, id := typesynth.Callee(info, expr.X)
		v, _ := info.Uses[id].(*types.Var)
		// A variable's own address is non-nil. Selectors on pointer
		// receivers can need evaluation and nil checks along the path.
		_, local := ast.Unparen(expr.X).(*ast.Ident)
		return expr.Op == token.AND && local && v != nil
	case *ast.CallExpr:
		_, id := typesynth.Callee(info, expr.Fun)
		fn, _ := info.Uses[id].(*types.Builtin)
		return fn != nil && fn.Name() == "new"
	}
	return false
}
