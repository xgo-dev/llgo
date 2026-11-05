package cl

import (
	"fmt"
	"go/types"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// llvmIntrinsicWrapper recognizes a single intrinsic call with scalar operands
// and results, scalar conversions, and tuple extraction/forwarding. Expansion
// also works across packages without LTO. The ordinary Go function remains
// available for function values, go and defer. No operation-specific rules or
// arbitrary Go inlining are involved.
func (p *context) llvmIntrinsicWrapper(fn *ssa.Function) *ssa.Function {
	if disableInline {
		return nil
	}
	if wrapper, ok := p.llvmWrappers[fn]; ok {
		return wrapper
	}
	original := fn
	if hasNoInlineDirective(fn) || p.llvmIntrinsicHasEnv(fn) {
		fn = nil
	}
	// Calls in importing packages still refer to the original Go SSA member.
	// Analyze the replacement body when a package patch supplies that member.
	if fn != nil && fn.Pkg != nil && fn.Signature.Recv() == nil {
		if patch, ok := p.patches[llssa.PathOf(fn.Pkg.Pkg)]; ok {
			if replacement := patch.Alt.Func(fn.Name()); replacement != nil {
				fn = replacement
			}
		}
	}
	if fn != nil && fn != original && (hasNoInlineDirective(fn) || p.llvmIntrinsicHasEnv(fn)) {
		fn = nil
	}
	if fn != nil && !p.isLLVMIntrinsicWrapper(fn) {
		fn = nil
	}
	if p.llvmWrappers == nil {
		p.llvmWrappers = make(map[*ssa.Function]*ssa.Function)
	}
	p.llvmWrappers[original] = fn
	return fn
}

func scalarIntrinsicType(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat) != 0
}

func scalarIntrinsicResults(results *types.Tuple) bool {
	if results.Len() == 0 {
		return false
	}
	for i := 0; i < results.Len(); i++ {
		if !scalarIntrinsicType(results.At(i).Type()) {
			return false
		}
	}
	return true
}

func (p *context) isLLVMIntrinsicWrapper(fn *ssa.Function) bool {
	sig := fn.Signature
	if len(fn.Blocks) != 1 || fn.Parent() != nil || len(fn.FreeVars) != 0 ||
		sig.Recv() != nil || sig.Variadic() || sig.TypeParams().Len() != 0 || fn.Origin() != nil ||
		!scalarIntrinsicResults(sig.Results()) {
		return false
	}
	if _, _, kind := p.funcName(fn); kind != goFunc {
		return false
	}
	known := make(map[ssa.Value]bool)
	for _, param := range fn.Params {
		if !scalarIntrinsicType(param.Type()) {
			return false
		}
		known[param] = true
	}
	valid := func(v ssa.Value) bool {
		if _, ok := v.(*ssa.Const); ok {
			return scalarIntrinsicType(v.Type())
		}
		return known[v]
	}
	calls := 0
	for _, instr := range fn.Blocks[0].Instrs {
		switch v := instr.(type) {
		case *ssa.Convert:
			if !valid(v.X) || !scalarIntrinsicType(v.Type()) {
				return false
			}
			known[v] = true
		case *ssa.ChangeType:
			if !valid(v.X) || !scalarIntrinsicType(v.Type()) {
				return false
			}
			known[v] = true
		case *ssa.Call:
			callee := v.Call.StaticCallee()
			if callee == nil || !scalarIntrinsicResults(v.Call.Signature().Results()) {
				return false
			}
			if _, _, kind := p.funcName(callee); kind != llvmIntrinsic {
				return false
			}
			for _, arg := range v.Call.Args {
				if !valid(arg) {
					return false
				}
			}
			calls++
			known[v] = true
		case *ssa.Extract:
			if !valid(v.Tuple) || !scalarIntrinsicType(v.Type()) {
				return false
			}
			known[v] = true
		case *ssa.Return:
			if calls != 1 || len(v.Results) != sig.Results().Len() {
				return false
			}
			if len(v.Results) == 1 {
				return valid(v.Results[0])
			}
			// Forward a complete intrinsic tuple without rebuilding an
			// aggregate or changing the Go result layout.
			first, ok := v.Results[0].(*ssa.Extract)
			if !ok || !types.Identical(first.Tuple.Type(), sig.Results()) {
				return false
			}
			for i, result := range v.Results {
				part, ok := result.(*ssa.Extract)
				if !ok || part.Tuple != first.Tuple || part.Index != i {
					return false
				}
			}
			return true
		case *ssa.DebugRef:
		default:
			return false
		}
	}
	return false
}

func (p *context) expandLLVMIntrinsicWrapper(b llssa.Builder, fn *ssa.Function, call *ssa.CallCommon) llssa.Expr {
	if len(call.Args) != len(fn.Params) {
		p.llvmIntrinsicError(call.Pos(), fn, fmt.Errorf("intrinsic wrapper parameter count mismatch: replacement requires %d arguments, call provides %d", len(fn.Params), len(call.Args)))
	}
	values := make(map[ssa.Value]llssa.Expr)
	args := p.compileValues(b, call.Args, fnNormal)
	for i, param := range fn.Params {
		values[param] = args[i]
	}
	value := func(v ssa.Value) llssa.Expr {
		if c, ok := v.(*ssa.Const); ok {
			return b.Const(c.Value, p.type_(c.Type(), llssa.GoDecl))
		}
		return values[v]
	}
	for _, instr := range fn.Blocks[0].Instrs {
		switch v := instr.(type) {
		case *ssa.Convert:
			values[v] = b.Convert(p.type_(v.Type(), llssa.GoDecl), value(v.X))
		case *ssa.ChangeType:
			values[v] = b.ChangeType(p.type_(v.Type(), llssa.GoDecl), value(v.X))
		case *ssa.Extract:
			values[v] = b.Extract(value(v.Tuple), v.Index)
		case *ssa.Call:
			callee := v.Call.StaticCallee()
			_, name, _ := p.funcName(callee)
			d := p.resolveLLVMIntrinsic(callee, name)
			operands := make([]llssa.Expr, len(v.Call.Args))
			for i, arg := range v.Call.Args {
				operands[i] = value(arg)
			}
			ret, err := b.CallLLVMIntrinsic(d, operands...)
			if err != nil {
				p.llvmIntrinsicError(call.Pos(), fn, err)
			}
			values[v] = ret
		case *ssa.Return:
			if len(v.Results) > 1 {
				return value(v.Results[0].(*ssa.Extract).Tuple)
			}
			return b.ChangeType(p.type_(fn.Signature.Results().At(0).Type(), llssa.GoDecl), value(v.Results[0]))
		}
	}
	panic("missing intrinsic wrapper return")
}
