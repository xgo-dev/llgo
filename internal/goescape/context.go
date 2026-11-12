package goescape

import (
	"go/types"

	"golang.org/x/tools/go/ssa"
)

func (f *functionAnalysis) boundValue(v ssa.Value) ssa.Value {
	if actual := f.bindings[v]; actual != nil {
		return actual
	}
	return v
}

// Specialization describes effects at a call site, never a new emitted body.
// A per-body budget includes recursive forwarding and is independent of
// unrelated program roots and archive-cache history.
func (f *functionAnalysis) contextSummary(fn *ssa.Function, args []ssa.Value) Summary {
	fallback := f.a.summary(fn)
	if !f.a.solved[fn] || f.contextDepth >= 4 || *f.contextBudget <= 0 || len(args) < len(fn.Params) {
		return fallback
	}
	var bindings map[ssa.Value]ssa.Value
	for i, param := range fn.Params {
		actual := f.boundValue(args[i])
		var targets []callTarget
		switch typ := param.Type().Underlying().(type) {
		case *types.Signature:
			targets = callTargets(&ssa.CallCommon{Value: actual})
		case *types.Interface:
			for j := 0; j < typ.NumMethods(); j++ {
				resolved := callTargets(&ssa.CallCommon{Value: actual, Method: typ.Method(j)})
				if len(resolved) == 0 {
					targets = nil
					break
				}
				targets = append(targets, resolved...)
			}
		}
		if len(targets) == 0 {
			continue
		}
		ready := true
		for _, target := range targets {
			ready = ready && f.a.solved[target.fn]
		}
		if !ready {
			continue
		}
		if bindings == nil {
			bindings = make(map[ssa.Value]ssa.Value)
		}
		bindings[param] = actual
	}
	if len(bindings) == 0 {
		return fallback
	}
	work := 0
	for _, block := range fn.Blocks {
		work += len(block.Instrs)
	}
	if work > 512 {
		return fallback
	}
	*f.contextBudget--
	summary, _ := f.a.analyzeBoundFunction(fn, bindings, f.contextDepth+1, f.contextBudget)
	return summary
}
