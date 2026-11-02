package cl

import (
	"go/token"
	"go/types"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// Keep this specialization within the existing dev Wasm DCE and linear-GC
// gates. Other collectors and debugger builds retain the public entry point.
func (p *context) directFinalizer(call *ssa.CallCommon) (obj, fn *ssa.MakeInterface, result types.Type, ok bool) {
	if !p.prog.DeadcodeDropEnabled() || !p.prog.GCRootsEnabled() || p.options.DebuggerFrames ||
		p.prog.Target() == nil || p.prog.Target().GOARCH != "wasm" {
		return
	}
	callee := call.StaticCallee()
	if callee == nil || callee.Pkg == nil ||
		callee.Name() != "SetFinalizer" || len(call.Args) != 2 {
		return
	}
	switch callee.Pkg.Pkg.Path() {
	case "runtime", "github.com/xgo-dev/llgo/runtime/internal/lib/runtime":
	default:
		return
	}
	obj, _ = call.Args[0].(*ssa.MakeInterface)
	if obj == nil {
		return
	}
	if _, pointer := obj.X.Type().Underlying().(*types.Pointer); !pointer {
		return
	}
	if nilFn, isConst := call.Args[1].(*ssa.Const); isConst && nilFn.IsNil() {
		ok = true
		return
	}
	fn, _ = call.Args[1].(*ssa.MakeInterface)
	if fn == nil {
		return
	}
	sig, function := fn.X.Type().Underlying().(*types.Signature)
	if !function || sig.Variadic() || sig.Params().Len() != 1 ||
		!types.Identical(obj.X.Type(), sig.Params().At(0).Type()) {
		return
	}
	// Native callback types and direct foreign functions do not necessarily
	// use the two-word Go closure ABI expected by the finalizer dispatcher.
	closure, goClosure := p.type_(fn.X.Type(), llssa.GoDecl).RawType().Underlying().(*types.Struct)
	if !goClosure || !llssa.IsClosure(closure) {
		return
	}
	if function, direct := fn.X.(*ssa.Function); direct {
		if _, _, kind := p.funcName(function); kind != goFunc {
			return
		}
	}
	switch sig.Results().Len() {
	case 0:
	case 1:
		result = sig.Results().At(0).Type()
		if _, aggregate := result.Underlying().(*types.Interface); !aggregate {
			return obj, fn, nil, false
		}
	default:
		return
	}
	ok = true
	return
}

func (p *context) finalizerDataOnly(value *ssa.MakeInterface) bool {
	refs, _ := nonDebugReferrers(value)
	if len(refs) == 0 {
		return false
	}
	for _, ref := range refs {
		call, normal := ref.(*ssa.Call)
		if !normal {
			return false
		}
		obj, fn, _, ok := p.directFinalizer(&call.Call)
		if !ok || value != obj && value != fn {
			return false
		}
	}
	return true
}

func (p *context) emitDirectFinalizer(b llssa.Builder, call *ssa.CallCommon, ds *explicitDeferStack, mayRecover bool) bool {
	_, fn, result, ok := p.directFinalizer(call)
	if !ok {
		return false
	}
	p.pkg.NeedRuntime = true
	objPtr := b.InterfaceData(p.compileValue(b, call.Args[0]))
	fnPtr := p.prog.Nil(p.prog.VoidPtr())
	if fn != nil {
		fnPtr = b.InterfaceData(p.compileValue(b, call.Args[1]))
	}
	var size uint64
	if result != nil {
		size = p.prog.SizeOf(p.type_(result, llssa.GoDecl))
	}
	const name = "runtime.llgoSetFinalizerPointer"
	helper := p.pkg.FuncOf(name)
	if helper == nil {
		sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(
			types.NewVar(token.NoPos, nil, "obj", types.Typ[types.UnsafePointer]),
			types.NewVar(token.NoPos, nil, "fn", types.Typ[types.UnsafePointer]),
			types.NewVar(token.NoPos, nil, "size", types.Typ[types.Uintptr]),
		), nil, false)
		helper = p.pkg.NewFuncEx(name, sig, llssa.GoDecl, false, false)
	}
	p.emitDo(b, llssa.Call, ds, mayRecover, helper.Expr, llssa.Builder.Call,
		objPtr, fnPtr, p.prog.IntVal(size, p.prog.Uintptr()))
	return true
}
