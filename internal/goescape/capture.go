package goescape

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// Copy only a small, initialized lexical variable whose address is never
// observed and whose closure reads it exclusively. Forwarded captures and
// method wrappers retain the ordinary shared-variable representation.
func immutableCapture(fn *ssa.Function, index int) bool {
	if fn.Parent() == nil || fn.Synthetic != "" {
		return false
	}
	v := fn.FreeVars[index]
	ptr, ok := v.Type().Underlying().(*types.Pointer)
	if !ok || captureWords(ptr.Elem(), 0) > 8 || v.Referrers() == nil {
		return false
	}
	for _, ref := range *v.Referrers() {
		switch use := ref.(type) {
		case *ssa.DebugRef:
		case *ssa.UnOp:
			if use.Op != token.MUL || use.X != v {
				return false
			}
		default:
			return false
		}
	}
	found := false
	for _, block := range fn.Parent().Blocks {
		for _, instr := range block.Instrs {
			closure, ok := instr.(*ssa.MakeClosure)
			if !ok || closure.Fn != fn {
				continue
			}
			found = true
			alloc, ok := closure.Bindings[index].(*ssa.Alloc)
			if !ok || alloc.Referrers() == nil {
				return false
			}
			var init *ssa.Store
			for _, ref := range *alloc.Referrers() {
				switch use := ref.(type) {
				case *ssa.DebugRef:
				case *ssa.Store:
					if init != nil || use.Addr != alloc {
						return false
					}
					init = use
				case *ssa.UnOp:
					if use.Op != token.MUL {
						return false
					}
				case *ssa.MakeClosure:
					if use.Fn != fn {
						return false
					}
					// The same variable in another capture position might be
					// writable through that free variable.
					for j, binding := range use.Bindings {
						if binding == alloc && j != index {
							return false
						}
					}
				default:
					return false
				}
			}
			if init == nil || !init.Block().Dominates(closure.Block()) {
				return false
			}
			if init.Block() == closure.Block() {
				initialized := false
				for _, instr := range init.Block().Instrs {
					if instr == init {
						initialized = true
					}
					if instr == closure && !initialized {
						return false
					}
				}
			}
		}
	}
	return found
}

// An upper bound in sixteen-byte units avoids depending on go/types sizes or
// introducing backend type caches into this representation-independent pass.
func captureWords(t types.Type, depth int) int {
	if depth > 8 {
		return 9
	}
	switch t := t.Underlying().(type) {
	case *types.Basic, *types.Pointer, *types.Chan, *types.Map, *types.Signature:
		return 1
	case *types.Slice:
		return 2
	case *types.Interface:
		return 1
	case *types.Array:
		if t.Len() > 8 {
			return 9
		}
		return int(t.Len()) * captureWords(t.Elem(), depth+1)
	case *types.Struct:
		n := 1 // Include possible trailing zero-field padding.
		for i := 0; i < t.NumFields() && n <= 8; i++ {
			n += captureWords(t.Field(i).Type(), depth+1)
		}
		return n
	}
	return 9
}
