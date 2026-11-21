package llvmfmv

import "github.com/xgo-dev/llvm"

// A helper outcome is an implication, not the value of a trusted CPU query.
// Never fold the call: other conditions, side effects and panics are observable.
func guardHelpers(candidates map[llvm.Value]bool, queries map[llvm.Value]uint64, avx2 *avx2Plan) map[llvm.Value][2]guardFact {
	helpers := make(map[llvm.Value][2]guardFact)
	var queue []llvm.Value
	pending := make(map[llvm.Value]bool)
	enqueue := func(fn llvm.Value) {
		if pending[fn] || fn.IsDeclaration() || fn.GlobalValueType().ReturnType() != fn.GlobalParent().Context().Int1Type() {
			return
		}
		// Don't derive facts from replaceable definitions.
		switch fn.Linkage() {
		case llvm.InternalLinkage, llvm.PrivateLinkage, llvm.ExternalLinkage:
		default:
			return
		}
		pending[fn] = true
		queue = append(queue, fn)
	}
	for fn := range candidates {
		enqueue(fn)
	}
	for len(queue) > 0 {
		fn := queue[0]
		queue = queue[1:]
		delete(pending, fn)
		blocks, facts := guardBlocks(fn, queries, helpers)
		sum := [2]guardFact{{impossible: true}, {impossible: true}}
		for bb, mask := range blocks {
			ret := bb.LastInstruction()
			if ret.InstructionOpcode() != llvm.Ret || ret.OperandsCount() != 1 {
				continue
			}
			for i := range sum {
				sum[i] = either(sum[i], together(guardFact{bits: mask}, facts.outcome(ret.Operand(0), i == 1)))
			}
		}
		// Expose only necessary CPU bits. Do not propagate constant return values
		// or treat a function with no returning path as a feature check.
		for i, v := range sum {
			if v.impossible {
				sum[i] = guardFact{}
			}
		}
		// Only add facts: recursive call graphs start with no evidence.
		old := helpers[fn]
		for i := range sum {
			sum[i].bits |= old[i].bits
		}
		if sum == old {
			continue
		}
		helpers[fn] = sum
		for use := fn.FirstUse(); !use.IsNil(); use = use.NextUse() {
			call := use.User()
			if call.IsACallInst().IsNil() || call.CalledValue() != fn {
				continue
			}
			parent := call.InstructionParent().Parent()
			if (avx2 != nil && !avx2.baselines[parent].IsNil()) || hasAttribute(parent, attributePrefix+"guarded") || !supported(parent) {
				continue
			}
			candidates[parent] = true
			enqueue(parent)
		}
	}
	return helpers
}
