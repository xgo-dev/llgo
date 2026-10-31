package ssa

import (
	"fmt"
	"go/types"
)

// DefineNativeExport supplies an ABI entry around a Go implementation. The
// same boundary is used by package-local exports and final-link wrappers.
func (p Package) DefineNativeExport(goName, nativeName string, sig *types.Signature, cc CallConv, ensureInit Function) Function {
	if sig.Recv() != nil || sig.Variadic() || sig.Results().Len() > 1 {
		panic(fmt.Errorf("native export %q has an unsupported signature", goName))
	}
	p.NeedRuntime = true
	implementation := p.NewFunc(goName, sig, GoDecl)
	enter := p.NewFunc(PkgRuntime+".EnterForeignThread", types.NewSignatureType(nil, nil, nil, nil,
		types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Bool])), false), GoDecl)
	exit := p.NewFunc(PkgRuntime+".ExitForeignThread", types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Bool])), nil, false), GoDecl)
	wrapper := p.NewFunc(nativeName, sig, cc)
	b := wrapper.MakeBody(1)
	if ensureInit != nil {
		b.Call(ensureInit.Expr)
	}
	registered := b.Call(enter.Expr)
	args := make([]Expr, sig.Params().Len())
	for i := range args {
		args[i] = wrapper.Param(i)
	}
	result := b.Call(implementation.Expr, args...)
	b.Call(exit.Expr, registered)
	if sig.Results().Len() == 0 {
		b.Return()
	} else {
		b.Return(result)
	}
	b.EndBuild()
	b.Dispose()
	return wrapper
}
