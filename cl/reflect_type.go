package cl

import (
	"go/types"

	"github.com/xgo-dev/llgo/internal/optlevel"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// Use the concrete descriptor only when all consumers are scalar queries.
// Each package proves this for its own SSA and needs no caller information.
func (p *context) directReflectTypeFactory(v *ssa.Call) *ssa.Function {
	if !p.prog.DeadcodeDropEnabled() || isRecoverTransparentWrapper(p.goFn) || p.options.DebuggerFrames || p.options.DebugSymbols || p.prog.Target().OptLevel != optlevel.Oz {
		return nil
	}
	fn := v.Call.StaticCallee()
	if fn == nil || fn.Pkg == nil || fn.Name() != "TypeOf" || !isReflectType(v.Type()) {
		return nil
	}
	patch := p.patches["reflect"]
	if patch.Alt == nil || fn.Pkg.Pkg != patch.Alt.Pkg && fn.Pkg.Pkg.Path() != "reflect" {
		return nil
	}
	factory := patch.Alt.Func("LLGoRTypeOf")
	if factory == nil {
		return nil
	}
	refs, ok := nonDebugReferrers(v)
	if !ok || len(refs) == 0 {
		return nil
	}
	for _, ref := range refs {
		use, ok := ref.(*ssa.Call)
		if !ok || use.Call.Value != v || directReflectTypeCallee(patch.Alt, use.Call.Method) == nil {
			return nil
		}
	}
	return factory
}

func directReflectTypeCallee(pkg *ssa.Package, method *types.Func) *ssa.Function {
	if method == nil || method.Pkg() == nil || method.Pkg() != pkg.Pkg && method.Pkg().Path() != "reflect" {
		return nil
	}
	sig := method.Type().(*types.Signature)
	if sig.Params().Len() != 0 || sig.Results().Len() != 1 {
		return nil
	}
	if _, scalar := sig.Results().At(0).Type().Underlying().(*types.Basic); !scalar {
		return nil
	}
	receiver := pkg.Type("rtype")
	if receiver == nil {
		return nil
	}
	obj, _, _ := types.LookupFieldOrMethod(types.NewPointer(receiver.Type()), true, pkg.Pkg, method.Name())
	implementation, ok := obj.(*types.Func)
	if !ok {
		return nil
	}
	return pkg.Prog.FuncValue(implementation)
}

func (p *context) emitDirectReflectTypeFactory(b llssa.Builder, v *ssa.Call, factory *ssa.Function) llssa.Expr {
	p.recordCallerLocationForCall(b, &v.Call)
	p.emitPCLineLabel(b, v.Pos())
	fn, _, _ := p.funcOf(factory)
	return b.Call(fn.Expr, p.compileValues(b, v.Call.Args, fnNormal)...)
}

func (p *context) tryDirectReflectTypeMethod(b llssa.Builder, call *ssa.CallCommon) (llssa.Expr, bool) {
	factory, ok := call.Value.(*ssa.Call)
	if !ok || p.directReflectTypeFactory(factory) == nil {
		return llssa.Nil, false
	}
	method := directReflectTypeCallee(p.patches["reflect"].Alt, call.Method)
	if method == nil {
		return llssa.Nil, false
	}
	receiver := p.compileValue(b, factory)
	b.AssertNilDeref(receiver)
	fn, _, _ := p.funcOf(method)
	return b.Call(fn.Expr, receiver), true
}
