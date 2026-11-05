package cl

import (
	"fmt"
	"go/types"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// llvmIntrinsicWrapper recognizes a single scalar intrinsic call surrounded
// only by scalar conversions. Expanding this small adapter at direct call sites
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

func (p *context) isLLVMIntrinsicWrapper(fn *ssa.Function) bool {
	sig := fn.Signature
	if len(fn.Blocks) != 1 || fn.Parent() != nil || len(fn.FreeVars) != 0 ||
		sig.Recv() != nil || sig.Variadic() || sig.TypeParams().Len() != 0 || fn.Origin() != nil ||
		sig.Results().Len() != 1 || !scalarIntrinsicType(sig.Results().At(0).Type()) {
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
			if callee == nil || !scalarIntrinsicType(v.Type()) {
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
		case *ssa.Return:
			return calls == 1 && len(v.Results) == 1 && valid(v.Results[0])
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
			return b.ChangeType(p.type_(fn.Signature.Results().At(0).Type(), llssa.GoDecl), value(v.Results[0]))
		}
	}
	panic("missing intrinsic wrapper return")
}
