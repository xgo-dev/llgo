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
	// MakeClosure and the Alloc/FreeVar nodes below always carry Referrers.
	refs, _ := nonDebugReferrers(closure)
	if len(refs) == 0 {
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
	if elem := alloc.Type().(*types.Pointer).Elem(); p.prog.SizeOf(p.type_(elem, llssa.GoDecl)) == 0 {
		return false
	}
	refs, _ := nonDebugReferrers(alloc)
	captured := false
	for _, ref := range refs {
		if captureStorageUse(ref, alloc) {
			continue
		}
		closure, ok := ref.(*ssa.MakeClosure)
		if !ok || !p.canStackAllocateClosure(closure) {
			return false
		}
		// MakeClosure.Fn is always a Function in Go SSA.
		callee := closure.Fn.(*ssa.Function)
		for i, binding := range closure.Bindings {
			if binding != alloc {
				continue
			}
			// Go SSA pairs each binding with the free variable at the same index.
			uses, _ := nonDebugReferrers(callee.FreeVars[i])
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
