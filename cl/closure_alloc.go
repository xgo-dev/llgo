package cl

import (
	"go/token"
	"go/types"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func (p *context) canStackAllocateClosure(closure *ssa.MakeClosure) bool {
	if p.prog.GCRootsEnabled() || closure.Block() == nil || closure.Parent().Recover != nil {
		// Explicit roots trace heap contexts, not pointer fields on the stack.
		return false
	}
	refs, ok := nonDebugReferrers(closure)
	if !ok || len(refs) == 0 {
		return false
	}
	for _, ref := range refs {
		call, ok := ref.(*ssa.Call)
		if !ok || call.Block() != closure.Block() || call.Call.Value != closure {
			return false
		}
		for _, arg := range call.Call.Args {
			if arg == closure {
				return false
			}
		}
	}
	return true
}

// A captured variable can use the caller's frame only if the address itself
// stays there. Loading or assigning its value is safe; returning/storing the
// address, passing it to a helper, and capturing it in an escaping closure
// all retain the conservative heap allocation.
func (p *context) canStackAllocateCapture(alloc *ssa.Alloc) bool {
	fn := alloc.Parent()
	if !alloc.Heap || fn == nil || fn.Recover != nil || alloc.Block() != fn.Blocks[0] || p.prog.GCRootsEnabled() {
		return false
	}
	if elem := alloc.Type().Underlying().(*types.Pointer).Elem(); p.prog.SizeOf(p.type_(elem, llssa.GoDecl)) == 0 {
		return false
	}
	refs, ok := nonDebugReferrers(alloc)
	if !ok {
		return false
	}
	captured := false
	for _, ref := range refs {
		if captureStorageUse(ref, alloc) {
			continue
		}
		closure, ok := ref.(*ssa.MakeClosure)
		if !ok || !p.canStackAllocateClosure(closure) {
			return false
		}
		callee, ok := closure.Fn.(*ssa.Function)
		if !ok {
			return false
		}
		for i, binding := range closure.Bindings {
			if binding != alloc {
				continue
			}
			uses, ok := nonDebugReferrers(callee.FreeVars[i])
			if !ok {
				return false
			}
			for _, use := range uses {
				if !captureStorageUse(use, callee.FreeVars[i]) {
					return false
				}
			}
		}
		captured = true
	}
	return captured
}

func captureStorageUse(instr ssa.Instruction, address ssa.Value) bool {
	switch instr := instr.(type) {
	case *ssa.UnOp:
		return instr.Op == token.MUL && instr.X == address
	case *ssa.Store:
		return instr.Addr == address && instr.Val != address
	}
	return false
}
