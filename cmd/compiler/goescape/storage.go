package goescape

import (
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/ssa"
)

// Use the same object inventory for solving and cache validation. A missing
// placement in an otherwise optimistic cached summary would be unsound.
func (a *analyzer) objectType(site ssa.Instruction) types.Type {
	switch v := site.(type) {
	case *ssa.Alloc:
		return v.Type().Underlying().(*types.Pointer).Elem()
	case *ssa.MakeInterface:
		indirect := !directInterface(v.X.Type())
		if a.config.IndirectInterface != nil {
			indirect = a.config.IndirectInterface(v.X.Type())
		}
		if indirect {
			return v.X.Type()
		}
	case *ssa.MakeClosure:
		fields := make([]*types.Var, len(v.Bindings))
		fn, _ := v.Fn.(*ssa.Function)
		for i, binding := range v.Bindings {
			typ := binding.Type()
			if fn != nil && a.result.CaptureByValue(fn.FreeVars[i]) {
				typ = typ.Underlying().(*types.Pointer).Elem()
			}
			fields[i] = types.NewVar(token.NoPos, nil, "capture"+strconv.Itoa(i), typ)
		}
		return types.NewStruct(fields, nil)
	case *ssa.MakeSlice, *ssa.MakeMap, *ssa.MakeChan:
		return site.(ssa.Value).Type()
	case *ssa.Convert:
		if copiedStringStorage(v.Type(), v.X.Type()) {
			return v.Type()
		}
	case *ssa.BinOp:
		if v.Op == token.ADD && stringType(v.Type()) {
			return v.Type()
		}
	}
	return nil
}
