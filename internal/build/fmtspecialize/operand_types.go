package fmtspecialize

import (
	"go/types"
	"slices"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func (plan *ValuePlan) recordBuiltin(call *ssa.Call, fn *ssa.Function, constantFormat bool) bool {
	if fn == nil {
		return false
	}
	args := formatArguments(call, call.Common().Args[1])
	// Constant formats with concrete operands already have the typed paths.
	hasInterface := slices.ContainsFunc(args, func(arg ssa.Value) bool {
		switch arg.(type) {
		case *ssa.ChangeInterface, *ssa.Parameter, *ssa.Phi:
			return true
		default:
			return false
		}
	})
	if len(args) == 0 || constantFormat && !hasInterface {
		return false
	}
	rewrite := formatRewrite{call: call, function: fn, builtinKinds: builtinOperandKinds(args, nil)}
	if rewrite.builtinKinds != 0 {
		plan.calls = append(plan.calls, rewrite)
	} else if hasInterface {
		plan.pending = append(plan.pending, rewrite)
	} else {
		return false
	}
	return true
}

func (plan *ValuePlan) resolveBuiltinCalls(roots []*ssa.Function) {
	if len(plan.pending) == 0 {
		return
	}
	incoming := directParameterInputs(plan.callee.Prog, roots)
	for _, rewrite := range plan.pending {
		// A scalar rewrite may already have replaced the original call.
		callee := rewrite.call.Common().StaticCallee()
		if callee != plan.callee && callee != plan.plainPrintf {
			continue
		}
		args := formatArguments(rewrite.call, rewrite.call.Common().Args[1])
		if kinds := builtinOperandKinds(args, incoming); kinds != 0 {
			rewrite.builtinKinds = kinds
			plan.calls = append(plan.calls, rewrite)
		}
	}
	plan.pending = nil
}

func builtinOperandKinds(args []ssa.Value, incoming map[*ssa.Parameter][]ssa.Value) uint32 {
	var kinds uint32
	for _, arg := range args {
		mask, known := builtinValueKinds(arg, incoming, make(map[ssa.Value]bool))
		if !known || mask == 0 {
			return 0
		}
		kinds |= mask
	}
	return kinds
}

// Walk backward from an interface use. A cycle contributes no new types.
// Every external input to the cycle still needs a concrete scalar source.
func builtinValueKinds(value ssa.Value, inputs map[*ssa.Parameter][]ssa.Value, seen map[ssa.Value]bool) (uint32, bool) {
	if seen[value] {
		return 0, true
	}
	seen[value] = true
	var sources []ssa.Value
	switch v := value.(type) {
	case *ssa.MakeInterface:
		if basic, ok := types.Unalias(v.X.Type()).(*types.Basic); ok {
			mask := builtinKindMask(basic.Kind())
			return mask, mask != 0
		}
		return 0, false
	case *ssa.ChangeInterface:
		return builtinValueKinds(v.X, inputs, seen)
	case *ssa.Phi:
		sources = v.Edges
	case *ssa.Parameter:
		sources = inputs[v]
	default:
		return 0, false
	}
	if len(sources) == 0 {
		return 0, false
	}
	var kinds uint32
	for _, source := range sources {
		mask, known := builtinValueKinds(source, inputs, seen)
		if !known {
			return 0, false
		}
		kinds |= mask
	}
	return kinds, true
}

// Only ordinary private functions in the executable with exclusively direct
// uses can receive parameter facts. Entry roots and address-taken functions
// can be called without an SSA call whose operands are available to inspect.
func directParameterInputs(prog *ssa.Program, roots []*ssa.Function) map[*ssa.Parameter][]ssa.Value {
	functions := ssautil.AllFunctions(prog)
	incoming := make(map[*ssa.Function][]*ssa.CallCommon)
	for fn := range functions {
		if fn.Pkg == nil || fn.Pkg.Pkg.Name() != "main" || fn.Parent() != nil ||
			fn.Object() == nil || fn.Object().Exported() || fn.Signature.Recv() != nil ||
			fn.TypeParams().Len() != 0 || len(fn.TypeArgs()) != 0 {
			continue
		}
		for _, param := range fn.Params {
			if types.IsInterface(param.Type()) {
				incoming[fn] = nil
				break
			}
		}
	}
	for _, root := range roots {
		delete(incoming, root)
	}
	var operands [8]*ssa.Value
	for fn := range functions {
		for _, block := range fn.Blocks {
			for _, instruction := range block.Instrs {
				var common *ssa.CallCommon
				if call, ok := instruction.(ssa.CallInstruction); ok {
					common = call.Common()
				}
				for _, ref := range instruction.Operands(operands[:0]) {
					if ref == nil {
						continue
					}
					callee, ok := (*ref).(*ssa.Function)
					calls, tracked := incoming[callee]
					if !ok || !tracked {
						continue
					}
					if common == nil || ref != &common.Value || common.StaticCallee() != callee ||
						fn.Pkg != callee.Pkg || len(common.Args) != len(callee.Params) {
						delete(incoming, callee)
						continue
					}
					incoming[callee] = append(calls, common)
				}
			}
		}
	}
	inputs := make(map[*ssa.Parameter][]ssa.Value)
	for fn, calls := range incoming {
		for i, param := range fn.Params {
			if types.IsInterface(param.Type()) {
				for _, call := range calls {
					inputs[param] = append(inputs[param], call.Args[i])
				}
			}
		}
	}
	return inputs
}
