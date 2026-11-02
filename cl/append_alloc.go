package cl

import (
	"go/types"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// canStackAllocateAppendArgs recognizes the temporary array built for
// append(dst, x, y, ...). Append copies its elements, never retaining the array.
// Restrict every use to the allocating block so a loop can reuse one stack
// slot without overlapping lifetimes. Do not change the shared Go SSA graph.
func (p *context) canStackAllocateAppendArgs(alloc *ssa.Alloc) bool {
	if !alloc.Heap || alloc.Comment != "varargs" || alloc.Block() == nil {
		return false
	}
	pointer, ok := alloc.Type().Underlying().(*types.Pointer)
	if !ok {
		return false
	}
	array, ok := pointer.Elem().Underlying().(*types.Array)
	if !ok || p.prog.SizeOf(p.type_(array, llssa.GoDecl)) == 0 {
		return false
	}
	if p.prog.GCRootsEnabled() && mayContainGCRoot(array) {
		// Explicit roots trace heap objects, not the contents of stack arrays.
		return false
	}
	refs, ok := nonDebugReferrers(alloc)
	if !ok {
		return false
	}
	appends := 0
	for _, ref := range refs {
		if ref.Block() != alloc.Block() {
			return false
		}
		switch ref := ref.(type) {
		case *ssa.IndexAddr:
			uses, ok := nonDebugReferrers(ref)
			if !ok {
				return false
			}
			for _, use := range uses {
				store, ok := use.(*ssa.Store)
				// Initializing a slot is safe; storing its address would escape.
				if !ok || store.Block() != alloc.Block() || store.Addr != ref || store.Val == ref {
					return false
				}
			}
		case *ssa.Slice:
			uses, ok := nonDebugReferrers(ref)
			if !ok || len(uses) != 1 {
				return false
			}
			call, ok := uses[0].(*ssa.Call)
			// The temporary is the copied source, never append's destination.
			if !ok || call.Block() != alloc.Block() || len(call.Call.Args) != 2 || call.Call.Args[1] != ref || call.Call.Args[0] == ref {
				return false
			}
			builtin, ok := call.Call.Value.(*ssa.Builtin)
			if !ok || builtin.Name() != "append" {
				return false
			}
			appends++
		default:
			return false
		}
	}
	// Require one consuming copy of the temporary's elements.
	return appends == 1
}
