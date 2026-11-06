package cl

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// reflectMethodFuncUnobserved proves that a Type.Method/MethodByName result
// only supplies metadata. Unknown uses retain the callable-method demand,
// including merely inspecting Func (IsValid, Interface, or a comparison).
// The proof runs before lowering obscures the Method's individual fields.
func reflectMethodFuncUnobserved(fn *ssa.Function, common *ssa.CallCommon) bool {
	if fn == nil || common == nil {
		return false
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if call, ok := instr.(*ssa.Call); ok && &call.Call == common {
				proof := reflectMethodUseProof{seen: make(map[ssa.Value]bool)}
				return proof.value(call)
			}
		}
	}
	return false
}

type reflectMethodUseProof struct {
	seen map[ssa.Value]bool
}

func (p *reflectMethodUseProof) value(v ssa.Value) bool {
	if p.seen[v] {
		return true
	}
	if len(p.seen) >= 128 || v.Referrers() == nil {
		return false
	}
	p.seen[v] = true
	if tuple, ok := v.Type().(*types.Tuple); ok {
		if tuple.Len() != 2 || !isReflectMethod(tuple.At(0).Type()) ||
			!types.Identical(tuple.At(1).Type(), types.Typ[types.Bool]) {
			return false
		}
		for _, use := range *v.Referrers() {
			switch use := use.(type) {
			case *ssa.DebugRef:
			case *ssa.Extract:
				if use.Index == 0 && !p.value(use) {
					return false
				}
			default:
				return false
			}
		}
		return true
	}
	if !isReflectMethod(v.Type()) {
		return false
	}
	for _, use := range *v.Referrers() {
		switch use := use.(type) {
		case *ssa.DebugRef:
		case *ssa.Field:
			if use.X != v || !isReflectMethodMetadataField(v.Type(), use.Field) {
				return false
			}
		case *ssa.Phi:
			if !p.value(use) {
				return false
			}
		case *ssa.Store:
			alloc, ok := use.Addr.(*ssa.Alloc)
			if use.Val != v || !ok || !p.storage(alloc) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (p *reflectMethodUseProof) storage(alloc *ssa.Alloc) bool {
	if p.seen[alloc] {
		return true
	}
	if len(p.seen) >= 128 || alloc.Referrers() == nil {
		return false
	}
	p.seen[alloc] = true
	ptr, ok := alloc.Type().Underlying().(*types.Pointer)
	if !ok || !isReflectMethod(ptr.Elem()) {
		return false
	}
	for _, use := range *alloc.Referrers() {
		switch use := use.(type) {
		case *ssa.DebugRef:
		case *ssa.Store:
			if use.Addr != alloc || use.Val == alloc {
				return false
			}
		case *ssa.UnOp:
			if use.Op != token.MUL || use.X != alloc || !p.value(use) {
				return false
			}
		case *ssa.FieldAddr:
			if use.X != alloc || !isReflectMethodMetadataField(ptr.Elem(), use.Field) ||
				!reflectMetadataFieldAddressUnescaped(use) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func reflectMetadataFieldAddressUnescaped(addr *ssa.FieldAddr) bool {
	if addr.Referrers() == nil {
		return false
	}
	for _, use := range *addr.Referrers() {
		switch use := use.(type) {
		case *ssa.DebugRef:
		case *ssa.UnOp:
			if use.Op != token.MUL || use.X != addr {
				return false
			}
		case *ssa.Store:
			if use.Addr != addr || use.Val == addr {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func isReflectMethod(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "reflect" && named.Obj().Name() == "Method"
}

func isReflectMethodMetadataField(t types.Type, index int) bool {
	st, ok := t.Underlying().(*types.Struct)
	return ok && index >= 0 && index < st.NumFields() && st.Field(index).Name() != "Func"
}
