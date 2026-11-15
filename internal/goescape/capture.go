package goescape

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

const maxCaptureCells = 64

// A capture's representation must agree across its whole lexical graph:
// copying an outer variable while an inner closure still retains its address
// would let that inner closure refer to a short-lived environment field.
type captureAnalysis struct {
	inspect func(*ssa.Function) bool
	sites   map[*ssa.Function][]*ssa.MakeClosure
	indexed map[*ssa.Function]bool
	proofs  map[*ssa.Alloc]bool
}

func (a *captureAnalysis) binding(fn *ssa.Function, index int) ssa.Value {
	parent := fn.Parent()
	if parent == nil || fn.Synthetic != "" || !a.inspect(fn) {
		return nil
	}
	if !a.indexed[parent] {
		a.indexed[parent] = true
		for _, block := range parent.Blocks {
			for _, instr := range block.Instrs {
				if closure, ok := instr.(*ssa.MakeClosure); ok {
					if child, ok := closure.Fn.(*ssa.Function); ok {
						a.sites[child] = append(a.sites[child], closure)
					}
				}
			}
		}
	}
	var binding ssa.Value
	for _, site := range a.sites[fn] {
		if index >= len(site.Bindings) || (binding != nil && binding != site.Bindings[index]) {
			return nil
		}
		binding = site.Bindings[index]
	}
	return binding
}

// Trace forwarded free variables to their initialized cell, then prove that
// every lexical consumer only loads or forwards it. Writes through a loaded
// pointer are allowed; writes to, or observations of, the variable's address
// retain shared storage. Method wrappers and opaque bodies remain excluded.
func (a *captureAnalysis) immutableCapture(fn *ssa.Function, index int) bool {
	for range maxCaptureCells {
		switch binding := a.binding(fn, index).(type) {
		case *ssa.Alloc:
			if proof, ok := a.proofs[binding]; ok {
				return proof
			}
			proof := a.immutableCell(binding)
			a.proofs[binding] = proof
			return proof
		case *ssa.FreeVar:
			fn = binding.Parent()
			index = -1
			for i, v := range fn.FreeVars {
				if v == binding {
					index = i
					break
				}
			}
			if index < 0 {
				return false
			}
		default:
			return false
		}
	}
	return false
}

func (a *captureAnalysis) immutableCell(alloc *ssa.Alloc) bool {
	if !a.inspect(alloc.Parent()) {
		return false
	}
	seen := map[ssa.Value]bool{alloc: true}
	pending := []ssa.Value{alloc}
	var init *ssa.Store
	var captures []*ssa.MakeClosure
	for len(pending) > 0 {
		cell := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		ptr, ok := cell.Type().Underlying().(*types.Pointer)
		if !ok || captureUnits(ptr.Elem(), 0) > 8 || cell.Referrers() == nil {
			return false
		}
		for _, ref := range *cell.Referrers() {
			switch use := ref.(type) {
			case *ssa.DebugRef:
			case *ssa.Store:
				if cell != alloc || init != nil || use.Addr != alloc {
					return false
				}
				init = use
			case *ssa.UnOp:
				if use.Op != token.MUL || use.X != cell {
					return false
				}
			case *ssa.MakeClosure:
				fn, ok := use.Fn.(*ssa.Function)
				if !ok || fn.Parent() != use.Parent() {
					return false
				}
				for i, binding := range use.Bindings {
					if binding != cell {
						continue
					}
					if i >= len(fn.FreeVars) || a.binding(fn, i) != cell {
						return false
					}
					v := fn.FreeVars[i]
					if !seen[v] {
						seen[v] = true
						if len(seen) > maxCaptureCells {
							return false
						}
						pending = append(pending, v)
					}
				}
				if cell == alloc {
					captures = append(captures, use)
				}
			default:
				return false
			}
		}
	}
	if init == nil || len(captures) == 0 || !captureInitializedOnce(alloc, init) {
		return false
	}
	for _, closure := range captures {
		if !init.Block().Dominates(closure.Block()) {
			return false
		}
		if init.Block() == closure.Block() {
			for _, instr := range init.Block().Instrs {
				if instr == init {
					break
				}
				if instr == closure {
					return false
				}
			}
		}
	}
	return true
}

// One SSA store can execute repeatedly. A capture is immutable only if every
// path back to its initializer creates a new instance of the variable first.
// Stop at the allocation, which precedes the store when both share a block.
func captureInitializedOnce(alloc *ssa.Alloc, init *ssa.Store) bool {
	allocation, initialization := alloc.Block(), init.Block()
	if allocation == initialization {
		return true
	}
	seen := make(map[*ssa.BasicBlock]bool)
	pending := append([]*ssa.BasicBlock(nil), initialization.Succs...)
	for len(pending) > 0 {
		block := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if block == allocation || seen[block] {
			continue
		}
		if block == initialization {
			return false
		}
		seen[block] = true
		pending = append(pending, block.Succs...)
	}
	return true
}

// A unit bounds sixteen bytes: a 64-bit slice needs two units (24 bytes),
// while an interface or function value fits one (16 bytes). Smaller scalars
// deliberately round up. No go/types size or backend type cache is required.
func captureUnits(t types.Type, depth int) int {
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
		return int(t.Len()) * captureUnits(t.Elem(), depth+1)
	case *types.Struct:
		n := 1 // Include possible trailing zero-field padding.
		for i := 0; i < t.NumFields() && n <= 8; i++ {
			n += captureUnits(t.Field(i).Type(), depth+1)
		}
		return n
	}
	return 9
}
