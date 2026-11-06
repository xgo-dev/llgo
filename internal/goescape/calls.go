package goescape

import (
	"go/types"

	"golang.org/x/tools/go/ssa"
)

type callTarget struct {
	fn   *ssa.Function
	args []ssa.Value
}

// Resolve a bounded set of syntactically proven function/interface values.
// Unknown phi inputs retain the conservative contract. SSA is not rewritten:
// lowering preserves interface checks, receiver evaluation and the target ABI.
func callTargets(call *ssa.CallCommon) []callTarget {
	var targets []callTarget
	seen := make(map[ssa.Value]bool)
	var visit func(ssa.Value) bool
	visit = func(v ssa.Value) bool {
		if seen[v] {
			return true
		}
		seen[v] = true
		if len(seen) > 32 {
			return false
		}
		switch v := v.(type) {
		case *ssa.ChangeType:
			return visit(v.X)
		case *ssa.ChangeInterface:
			return visit(v.X)
		case *ssa.TypeAssert:
			if _, ok := v.AssertedType.Underlying().(*types.Interface); !ok || v.CommaOk {
				return false
			}
			return visit(v.X)
		case *ssa.Extract:
			if assertion, ok := v.Tuple.(*ssa.TypeAssert); ok && v.Index == 0 {
				if _, ok := assertion.AssertedType.Underlying().(*types.Interface); ok {
					return visit(assertion.X)
				}
			}
			return false
		case *ssa.Phi:
			for _, e := range v.Edges {
				if !visit(e) {
					return false
				}
			}
			return true
		case *ssa.Function:
			if call.IsInvoke() {
				return false
			}
			targets = append(targets, callTarget{v, append([]ssa.Value(nil), call.Args...)})
			return true
		case *ssa.MakeClosure:
			if call.IsInvoke() {
				return false
			}
			fn, ok := v.Fn.(*ssa.Function)
			if !ok {
				return false
			}
			args := append([]ssa.Value(nil), call.Args...)
			args = append(args, v.Bindings...)
			targets = append(targets, callTarget{fn, args})
			return true
		case *ssa.MakeInterface:
			if !call.IsInvoke() {
				return false
			}
			if _, open := v.X.Type().Underlying().(*types.Interface); open {
				return false
			}
			selection := types.NewMethodSet(v.X.Type()).Lookup(call.Method.Pkg(), call.Method.Name())
			if selection == nil {
				return false
			}
			fn := v.Parent().Prog.MethodValue(selection)
			if fn == nil {
				return false
			}
			args := append([]ssa.Value{v.X}, call.Args...)
			targets = append(targets, callTarget{fn, args})
			return true
		}
		return false
	}
	if !visit(call.Value) || len(targets) == 0 {
		return nil
	}
	return targets
}

// A known callback need not publish all captures. Include both heap and result
// leaks: an unknown caller can retain any value returned by the callback.
func (f *functionAnalysis) callback(value ssa.Value) bool {
	if _, ok := value.Type().Underlying().(*types.Signature); !ok {
		return false
	}
	targets := callTargets(&ssa.CallCommon{Value: value})
	if targets == nil {
		return false
	}
	for _, target := range targets {
		s := f.a.summary(target.fn)
		first := len(s.Parameters) - len(target.fn.FreeVars)
		if first < 0 || len(target.args) != len(target.fn.FreeVars) {
			return false
		}
		for i, binding := range target.args {
			p := s.Parameters[first+i]
			f.effect(&f.g.heap, binding, p.Heap)
			f.effect(&f.g.mutator, binding, p.Mutator)
			if p.Callee != 0 {
				f.flow(&f.g.heap, binding, int(p.Callee))
			}
			for _, result := range p.Results {
				f.effect(&f.g.heap, binding, result)
			}
		}
	}
	return true
}
