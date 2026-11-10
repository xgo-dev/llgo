package cl

import "golang.org/x/tools/go/ssa"

// nonDebugReferrers returns the executable users of v. DebugRef is an SSA
// pseudo-instruction and must not change lowering decisions based on use count.
func nonDebugReferrers(v ssa.Value) (refs []ssa.Instruction, available bool) {
	all := v.Referrers()
	if all == nil {
		return nil, false
	}
	for _, ref := range *all {
		if _, ok := ref.(*ssa.DebugRef); !ok {
			refs = append(refs, ref)
		}
	}
	return refs, true
}

// singleNonDebugReferrer returns the sole executable user without allocating.
// Missing referrer information and any other use count return nil.
func singleNonDebugReferrer(v ssa.Value) ssa.Instruction {
	all := v.Referrers()
	if all == nil {
		return nil
	}
	var sole ssa.Instruction
	for _, ref := range *all {
		if _, debug := ref.(*ssa.DebugRef); debug {
			continue
		}
		if sole != nil {
			return nil
		}
		sole = ref
	}
	return sole
}
